# Lemon Squeezy activation brief

Branch: `fix/lemonsqueezy-live-activation` (based on `origin/main`)
Status: **verified end to end against Lemon Squeezy test mode on 2026-07-27.** Not deployed.

---

## 1. Why this exists

The Lemon Squeezy code path was written while the store was pending approval and had
**never executed against a real API response**. Activating it was treated as shipping
untested code, not as a config change.

An audit against the current Lemon Squeezy API docs found nine defects. Four would have
broken live checkouts or corrupted subscription rows.

---

## 2. What was wrong, and what changed

### Checkout could never have worked

| Defect | Fix |
|---|---|
| `CheckoutResult` had no JSON tags, so the API returned `{"RedirectURL":…}` while the UI reads `redirect_url`. `window.location.href` was set to `undefined`. | Added `json:"redirect_url"` / `json:"access_code,omitempty"` |
| `store_id` and `variant_id` were sent in `data.attributes`. They are **read-only response fields** — the only valid create attributes are `custom_price`, `product_options`, `checkout_options`, `checkout_data`, `preview`, `test_mode`, `expires_at`. | Removed from attributes; store and variant are addressed via `relationships` only |

### Webhooks corrupted the subscription row

`normaliseLSEvent` ended in `default: return e` and `HandleWebhook` returned a populated
event for **every** event type. The normal signup flow fires three webhooks:

```
order_created  →  subscription_created  →  subscription_payment_success
```

`order_created` carries an **Order**, not a Subscription. It has `custom_data` (so it was
not skipped) but no top-level `variant_id` and no `renews_at`. It wrote:

- `plan` = `free` (from `planFromVariant(0)`)
- `status` = `paid`
- `provider_sub_id` = the **order** ID, breaking cancel and portal

`order_created` and `subscription_created` are delivered concurrently, so whichever landed
last won — customers would non-deterministically end up on `free` moments after paying.

**Fix:** only seven `subscription_*` events are processed; everything else returns `nil`.

### Annual subscribers were recorded as monthly

`first_subscription_item.interval` **does not exist**. The object contains only `id`,
`subscription_id`, `price_id`, `quantity`, `created_at`, `updated_at` — and it is **`null`
while the subscription is on a free trial**, which is our default flow.

**Fix:** interval is derived from `variant_id`, which we already map for the plan.

### Trials never displayed

`WebhookEvent.TrialEnd` was never populated, and `processWebhookEvent` never copied it to
the model. Because the upsert does `trial_end = excluded.trial_end`, every webhook wrote
NULL over it. Result: the Pro card showed "Trial ends soon" forever, and the trial-expiry
check in `GetSubscription` (which requires `TrialEnd != nil`) could never fire.

**Fix:** `trial_ends_at` is parsed and plumbed through. Events with no trial date now carry
the stored value forward instead of nulling it — this also stops Paystack webhooks wiping
what `VerifyPaystack` recorded.

### Expired subscriptions kept Pro forever

`normaliseLSStatus` began with `if cancelled { return "active" }`, before the status
switch. `cancelled` stays `true` after a subscription expires, so `status: "expired"` was
normalised to `active`, `IsActive` returned true, and access never ended.

**Fix:** `expired` / `unpaid` / `past_due` are evaluated before the `cancelled` short-circuit.

### Cancelling cut off access early

Lemon Squeezy fires `subscription_cancelled` at the **moment of cancellation**, but the
customer keeps access until `ends_at`. The code mapped it straight to
`subscription.canceled`, which sets `plan = "free"` immediately.

**Fix:** `subscription_cancelled` → `subscription.updated` (sets `cancel_at_period_end`,
`current_period_end = ends_at`, plan stays `pro`). `subscription_expired` does the
downgrade. Paystack's mapping is unchanged.

### Failures were hidden

- `GetPortalURL` fell back to `returnURL + "/settings/billing"` when the portal URL was
  missing — a no-op redirect with no error anywhere. **Now returns an error.**
- A failed LS cancel was logged `(non-fatal)` and the response still said
  `cancelled: true` — customer sees "cancelled" while their card is still charged.
  **Now returns 502 and leaves the row untouched for a retry.**
- Webhook processing errors returned 200, so LS never retried. **Now returns 500.**

### Security

An empty `LEMONSQUEEZY_WEBHOOK_SECRET` still produces a valid HMAC, so anyone who guessed
the secret was unset could forge webhooks and grant themselves Pro. There was no startup
validation. **Now rejects webhooks when the secret or the header is empty, and warns at
boot when any billing config is unset.**

### Verified correct — deliberately unchanged

- `checkout_data.custom` as an object; webhook read at `meta.custom_data.user_id` as a string
- `X-Signature` header, HMAC-SHA256, hex digest, over the raw body
- `on_trial` → `trialing`
- `GET /v1/customers/{id}` → `data.attributes.urls.customer_portal`
- `ProviderCustomerID` holds the LS numeric **customer** ID (not order or subscription)
- `DELETE /v1/subscriptions/{id}` with `ProviderSubID`
- The portal URL is pre-signed and expires after 24h, so it is fetched per request and
  must **never** be cached

---

## 3. Commits

```
f2e44bc  fix(billing): align Lemon Squeezy provider with the live API
a7d3425  fix(billing): persist trial_end and stop hiding webhook failures
091639b  test(billing): cover Lemon Squeezy webhook parsing and signatures
2ac20e7  fix(ui): type subscription provider as lemonsqueezy
```

Each builds and tests green independently — the history is bisectable.

Files: `internal/billing/lemonsqueezy.go`, `internal/billing/provider.go`,
`internal/handler/billing.go`, `internal/store/store.go`, `main.go`,
`ui/src/types/index.ts`, `internal/billing/lemonsqueezy_test.go` (new).

---

## 3a. READ THIS BEFORE DEPLOYING — variant ID vs product ID

The test run failed on the very first API call with:

```
404 {"detail":"The related resource does not exist.",
     "source":{"pointer":"/data/relationships/variant"}}
```

Cause: `LEMONSQUEEZY_VARIANT_PRO_MONTHLY` and `_ANNUAL` held **product IDs**, not
variant IDs. In this store the two are numerically similar enough to look right:

| | product ID | variant ID |
|---|---|---|
| Pro monthly ($7/mo) | `1249619` | **`1953470`** |
| Pro annual ($67/yr) | `1249625` | **`1953478`** |

Product IDs are in the `1249xxx` range, variant IDs in the `195xxxx` range.

**The live values previously in `.env.local` were `124926…` and `124927…` — also the
`1249xxx` product range.** So production is very likely carrying the same mistake, and
live checkout will 404 exactly as this did. Before deploying, pull the real variant IDs
with a **live** API key:

```bash
curl -s https://api.lemonsqueezy.com/v1/variants \
  -H "Authorization: Bearer $LIVE_KEY" -H 'Accept: application/vnd.api+json' \
  | jq -r '.data[] | "\(.id)  product=\(.attributes.product_id)  \(.attributes.interval)"'
```

In the dashboard the variant ID is the number in the URL when you open the *variant*,
not the product.

## 4. Before testing — dashboard setup

Lemon Squeezy test mode is a **fully separate data space**. A test-mode API key only sees
test-mode data, and products copied to live mode get **new variant IDs**. The live
credentials cannot drive a test run.

With the test-mode toggle ON:

1. Copy the Pro product to test mode. **Record both new variant IDs.**
2. Confirm the 14-day trial carried over onto both variants. Without it, `on_trial` never
   fires and the trial path is untested.
3. Create a **test-mode API key**.
4. Create a **test-mode webhook** with its own signing secret, pointed at your tunnel.
   Subscribe to exactly: `subscription_created`, `subscription_updated`,
   `subscription_cancelled`, `subscription_expired`, `subscription_resumed`,
   `subscription_paused`, `subscription_unpaused`.
   Do **not** subscribe to `order_created` or `subscription_payment_*`.

Tunnel: `cloudflared tunnel --url http://localhost:8080`
Webhook URL: `https://<tunnel>/billing/webhook/lemonsqueezy`

Set all five `LEMONSQUEEZY_*` values in `.env.local` to the test set, plus
`LEMONSQUEEZY_TEST_MODE=true`. (`.env.local` is gitignored; secrets never enter the repo.)

---

## 4a. Test-mode results — 2026-07-27

Run against store `384187` (Hookdrop) in test mode, tunnelled via ngrok, with real
checkouts completed on card `4242 4242 4242 4242`.

| flow | result |
|---|---|
| `CreateCheckout` monthly + annual | hosted checkout URL returned; response key is `redirect_url` |
| Trial subscription | `plan=pro`, `status=trialing`, `trial_end` populated (+14 days) |
| Signature verification | every delivery verified; forged signature → 400 |
| `provider` column | `lemonsqueezy` — not an event-type string |
| IDs stored | `provider_customer_id=9487943`, `provider_sub_id=2381221` — both numeric LS IDs |
| **Annual interval** | **`interval=year`** — the field the payload cannot supply |
| `GetPortalURL` | real pre-signed portal URL with `test_mode=1` |
| `CancelSubscription` | `DELETE` succeeded; row kept `plan=pro`, `cancel_at_period_end=1`, access until trial end |
| Webhook dedupe | payload replayed twice → 200 each, zero new rows, logged as duplicate |

Not covered by the live run, and why:

- **`order_created` fall-through.** The dashboard webhook subscribes only to the seven
  `subscription_*` events, so LS never sent it. Covered by unit tests. Not subscribing is
  the safer configuration anyway.
- **Trial expiry / `subscription_expired`.** Needs a 14-day wait or a dashboard-forced
  expiry.

## 5. Test sequence (to repeat)

**Boot** — `go run .` — check the startup log for the unset-config warning and the
`LEMONSQUEEZY_TEST_MODE=true` line.

**Checkout** — `/settings/billing` → USD Pro button. Expect a redirect to
`*.lemonsqueezy.com/checkout/custom/…`, **not** `/undefined`. A 422 now includes the LS
response body naming the offending field.

**Pay** — card `4242 4242 4242 4242`, exp `12/35`, CVC `123`.

**Watch** — `docker logs -f hookdrop`. Expect `order_created` and
`subscription_payment_success` to be silently ignored, then:

```
processWebhookEvent: user=<uuid> plan=pro provider=lemonsqueezy status=trialing interval=month trial_end=... cancel_at_end=false
```

Expect **absent**: `missing user_id`, `plan=free`, `status=paid`, `invalid signature`.

**Verify the row:**

```sql
SELECT user_id, plan, provider, status, currency, interval,
       provider_customer_id, provider_sub_id,
       trial_end, current_period_end, cancel_at_period_end, updated_at
FROM subscriptions
WHERE user_id = '<test user uuid>';
```

All must hold:

- `plan` = `pro`
- `provider` = **`lemonsqueezy`** — not an event-type string (the exact bug hit with Paystack)
- `status` = `trialing`
- `interval` = `month`; **then repeat on annual and confirm `year`** — the only way to catch
  the interval regression
- `trial_end` **populated**, ≈ now + 14 days
- `current_period_end` populated
- `provider_customer_id` = short **numeric** string matching the LS Customers page — not a
  UUID, order ID, or subscription ID
- `provider_sub_id` = numeric subscription ID from the LS Subscriptions page
- `cancel_at_period_end` = `0`

UI check: the Pro card reads **"Trial ends {date}"**, not "Trial ends soon".

**Portal** — Manage subscription → redirects to `*.lemonsqueezy.com/billing?expires=…`.
A 500 now means the call genuinely failed; the silent bounce-back is gone.

**Cancel** — LS dashboard shows Cancelled with an `ends_at`. Re-run the SQL:
`cancel_at_period_end` = `1`, `current_period_end` = `ends_at`, and **`plan` still `pro`**.
UI shows "Access until {date}".
To exercise expiry without waiting, set `current_period_end` to a past timestamp and
confirm the read-time auto-expiry reports `plan=free`.

**Signature** — replay a delivery with the secret temporarily wrong; expect
`signature invalid` and a 400. Blank the secret entirely; expect everything rejected.

---

## 6. Deploy

Only after the full flow passes locally.

Paste into `/opt/hookdrop/docker-compose.yml` under the app service's `environment:`,
using the **live** values:

```yaml
      - LEMONSQUEEZY_API_KEY=<live api key>
      - LEMONSQUEEZY_STORE_ID=<live store id>
      - LEMONSQUEEZY_WEBHOOK_SECRET=<live webhook signing secret>
      - LEMONSQUEEZY_VARIANT_PRO_MONTHLY=<live monthly variant id>
      - LEMONSQUEEZY_VARIANT_PRO_ANNUAL=<live annual variant id>
      - LEMONSQUEEZY_TEST_MODE=false
```

Register the live webhook at `https://api.hookdrop.app/billing/webhook/lemonsqueezy` with
the same seven events. Then `scripts/deploy.sh`, and run one real checkout with a real
card that you refund from the dashboard afterwards.

Rollback: revert the four commits and redeploy. No schema migration was involved, so no
data migration is needed — `GetSubscriptionByProviderSubID` only reads.

---

## 7. Known issues, deliberately out of scope

- The `billing_events` table is defined but never written to. There is **no webhook
  idempotency and no audit trail**; duplicate or out-of-order deliveries are last-writer-wins.
- `VerifyPaystack` grants Pro even when Paystack verification fails outright, and does not
  check that the transaction belongs to the calling user. This is a free-Pro escalation on
  the **Paystack** path and should be fixed separately.
- `billing.Limits` has no JSON tags, so `/billing/subscription` returns
  `{"MaxNamedEndpoints":…}` while the TS type expects `max_named_endpoints`. Latent — no UI
  code reads it today.
- `ProviderForCurrency` / `ProviderForCountry` still return `"stripe"` and have no callers.
