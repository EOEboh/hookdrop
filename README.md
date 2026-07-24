# hookdrop

Hookdrop is a hosted tool for capturing, inspecting, and forwarding webhooks. Point
any webhook provider at your hookdrop inbox URL, watch requests arrive live in the
dashboard or your terminal, verify their signatures, replay them, and forward each
one to a local server the provider cannot reach on its own.

Website: [hookdrop.app](https://hookdrop.app)

## Why hookdrop

Webhook providers such as Stripe, GitHub, and Paystack send HTTP requests to a public
URL, which makes them awkward to develop against. Hookdrop gives you a public inbox
URL, captures every request in full, and lets you:

- See exactly what was sent: method, headers, raw body, size, source IP, and timestamp.
- Keep a searchable history of past deliveries instead of losing them.
- Verify provider signatures so you know a request is genuine.
- Replay any captured request to a target of your choice.
- Forward live webhooks to `localhost` while you build, using the CLI.

## Features

- **Public webhook capture** at `/i/<endpoint>` for any HTTP method, with the full
  request recorded.
- **Two endpoint types**: throwaway temporary sessions that expire after 24 hours,
  and persistent named endpoints with a slug you choose.
- **Live streaming** over Server-Sent Events to both the web dashboard and the CLI.
- **Signature verification** with built-in schemes for Stripe, GitHub, and Paystack,
  plus a generic HMAC fallback. Each request is tagged verified, failed, or unverified.
- **Replay** a captured request to any URL, with an optional body override.
- **History and filtering** of past requests per endpoint.
- **Passwordless auth** using email magic links.
- **API tokens** for the CLI and programmatic access, stored only as hashes.
- **Free and Pro plans**, billed through LemonSqueezy or Paystack.

## Quickstart with the CLI

The CLI streams your captured webhooks into the terminal and can forward each one to
a local server.

### Install

macOS and Linux, via the install script:

```sh
curl -fsSL https://raw.githubusercontent.com/EOEboh/hookdrop/main/scripts/install.sh | sh
```

macOS and Linux, via Homebrew:

```sh
brew tap EOEboh/hookdrop
brew trust eoeboh/hookdrop
brew install hookdrop
```

Windows: download the archive from the
[releases page](https://github.com/EOEboh/hookdrop/releases).

### Use

```sh
hookdrop login                                  # authenticate in your browser
hookdrop whoami                                 # show account, plan, and limits
hookdrop endpoints                              # list your named endpoints
hookdrop listen                                 # stream an endpoint live
hookdrop listen my-endpoint                     # stream a specific endpoint
hookdrop listen my-endpoint -f 3000             # forward each webhook to localhost:3000
hookdrop listen my-endpoint -f 3000 --path /webhook   # forward to a specific path
hookdrop logout                                 # remove the stored token
```

Forwarded requests carry `X-Hookdrop-Forwarded: true` and `X-Hookdrop-Original-Id`
headers, and the stream reconnects automatically with backoff if the connection drops.

See [cli/README.md](cli/README.md) for the full CLI reference.

## Architecture

Hookdrop has three parts:

1. **API server** (Go, `main.go` and `internal/`). A single binary built on the
   standard library `net/http`. It is both the public webhook receiver and the
   authenticated backend for the dashboard and CLI. Data is stored in SQLite.
2. **Dashboard** (`ui/`). A React and TypeScript single-page app with the live
   request feed, request detail, endpoint and secret management, replay, and billing.
3. **CLI** (`cli/`). A separate Go module that authenticates, subscribes to the same
   live stream, and forwards webhooks to a local target.

A webhook flows through the system like this: a provider sends a request to
`/i/<endpoint>`, the server records it and runs signature verification if secrets are
configured, the request is written to SQLite, and it is broadcast over SSE so any
connected dashboard or CLI session sees it instantly. History is read back through the
API, and retention depends on the plan.

## Tech stack

- Backend: Go 1.23, standard library `net/http`, SQLite in WAL mode.
- Frontend: React 19, TypeScript, Vite, Tailwind CSS v4.
- CLI: Go with cobra, released with goreleaser to GitHub Releases and a Homebrew tap.
- Email: Resend for magic links.
- Billing: LemonSqueezy and Paystack.

## Local development

Requirements: Go 1.23 or newer, Node.js with pnpm.

### Backend

The server reads configuration from a `.env.local` file in the working directory and
falls back to defaults for anything not set.

```sh
go run ./main.go
```

It listens on `:8080` by default and stores data in `hookdrop.db`.

Common settings:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `DB_PATH` | `hookdrop.db` | SQLite database path |
| `ALLOWED_ORIGIN` | `http://localhost:5173` | CORS origin |
| `API_URL` | `http://localhost:8080` | Base API URL |
| `FRONTEND_URL` | `http://localhost:5173` | Dashboard URL |
| `JWT_SECRET` | `change-me-in-production` | Session and token signing secret |
| `RESEND_API_KEY` | empty | Magic-link email delivery |

Billing requires additional LemonSqueezy and Paystack variables. See `main.go` for
the complete list.

### Dashboard

```sh
cd ui
pnpm install
pnpm dev
```

The dev server runs on `http://localhost:5173`, which matches the backend defaults.

### CLI against a local backend

```sh
cd cli
CGO_ENABLED=0 go build -o hookdrop .
HOOKDROP_API_URL=http://localhost:8080 ./hookdrop login
```

## Releasing

CLI releases are cut locally with goreleaser. See [RELEASING.md](RELEASING.md).
