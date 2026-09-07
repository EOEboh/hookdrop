# Running hookdrop on Render + Turso

The free path: Render runs the container, Turso holds the database. Neither
asks for a credit card.

This exists because Render's free tier **wipes the filesystem** on every
spin-down, restart and redeploy. Sleeping is survivable for this app; losing
the database is not. Moving the data to Turso is what makes a free, sleeping
platform safe — and as a side effect the app becomes stateless, so moving
hosts again later costs almost nothing.

## What changes

`store.New` dispatches on the DSN: a `libsql://`, `http(s)://` or `ws(s)://`
path opens the libSQL driver; anything else is the local-file driver exactly as
before. Local development is unaffected.

The image is built `CGO_ENABLED=0`, so it is pure Go, ~7 MB, and cross-compiles
to any architecture for free. **That image cannot open a local file** — the
local driver needs cgo. Given a file path it fails at boot with
`go-sqlite3 requires cgo to work. This is a stub`, which is loud and obvious
rather than silent.

## 1. Import the database into Turso

Sign up at [turso.tech](https://turso.tech) — no card. Then, from a machine
holding a **consistent snapshot** of production (see the caveat below):

```sh
curl -sSfL https://get.tur.so/install.sh | bash
turso auth signup

turso db create hookdrop --from-file ./hookdrop.db
turso db show hookdrop --url                 # libsql://hookdrop-<org>.turso.io
turso db tokens create hookdrop              # the auth token
```

Take the snapshot with `.backup`, not `cp`. The production database is in WAL
mode with a live writer, so a copied file is missing whatever sits in the WAL:

```sh
ssh deploy@<prod> "sqlite3 /opt/hookdrop/data/hookdrop.db \
  \".timeout 10000\" \".backup /tmp/snap.db\""
scp deploy@<prod>:/tmp/snap.db ./hookdrop.db
```

Verify the import landed before trusting it:

```sh
turso db shell hookdrop "SELECT count(*) FROM users;"          # expect 82
turso db shell hookdrop "SELECT count(*) FROM subscriptions;"
```

`DB_PATH` is then the URL and token joined:

```
libsql://hookdrop-<org>.turso.io?authToken=<token>
```

## 2. Create the Render service

Dashboard → **New → Blueprint** → connect this repo. It reads `render.yaml`
and prompts for the twelve secret values; take them from the old host's
compose file, not from `.env.local`, which mixes live and test credentials.

Render injects `PORT`. Do not set it.

Verify on the `.onrender.com` URL Render assigns, before touching DNS:

```sh
curl -s https://hookdrop-api-XXXX.onrender.com/health
```

The startup log must show no `WARNING: unset config` line.

## 3. Keep it awake

Render's free tier sleeps after 15 idle minutes and takes ~1 minute to wake.
For a webhook-capture product that means providers time out and log failed
deliveries — the one part of "free" that actually degrades the product.

Point a free uptime monitor (UptimeRobot or cron-job.org, neither needs a card)
at `/health` every 10 minutes and it never sleeps.

Budget note: the free allowance is **750 instance-hours per month** and a full
month is 744. One always-awake service fits, with 6 hours to spare — so this
works, but a second always-on free service will not.

## 4. Cut over

`api.hookdrop.app` does not change: it is compiled into the released CLI and
registered as the webhook URL at both billing providers.

1. Add `api.hookdrop.app` in Render (already declared in `render.yaml`) and
   note the CNAME target it gives you.
2. Re-export and re-import the database at the last moment, so the copy in
   Turso is current: stop the container on the old host (a clean shutdown
   checkpoints the WAL), snapshot, and import again.
3. Point the old host's nginx at the Render URL, so anything still resolving
   to the old IP is proxied rather than dropped:

   ```nginx
   location / {
       proxy_pass https://hookdrop-api-XXXX.onrender.com;
       proxy_set_header Host hookdrop-api-XXXX.onrender.com;
       proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       proxy_buffering off;
       proxy_read_timeout 3600s;
   }
   ```

   `proxy_buffering off` and the long read timeout keep SSE alive through the
   hop. The app reads the first `X-Forwarded-For` entry, so per-IP rate
   limiting and the recorded payer IP survive.

4. Change the DNS record for `api.hookdrop.app` to Render's target.
5. Send a test webhook from the Lemon Squeezy **and** Paystack dashboards. A
   secret mistyped in step 2 surfaces here as `signature invalid`, rather than
   during a real payment.
6. Once the old host's access log is quiet, decommission it.

## Known trade-offs

- **Latency.** Every query is a network round trip instead of a local file
  read. The capture path makes four store calls per webhook, several issuing
  more than one statement, so expect roughly 150-400 ms added per captured
  request against a cloud database. Fine for capture, more noticeable on
  dashboard loads.
- **Turso free limits:** 5 GB, 500M row reads and 10M row writes per month.
  Not close to binding at current volume — but each captured webhook is a
  write, so this scales with traffic, not users.
- **Backups.** `scripts/backup.sh` targets a local file and does not apply
  here. Turso keeps its own point-in-time backups; `turso db shell hookdrop
  .dump` gives you an off-platform copy, and it is worth running on a schedule
  rather than trusting the platform alone.
- **`-repair-paystack`** runs against `DB_PATH`, so it works unchanged — point
  it at the Turso URL.
