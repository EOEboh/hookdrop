# hookdrop CLI

Stream webhooks captured by [hookdrop](https://hookdrop.app) into your
terminal, and forward each one to a local server the hosted backend can't
reach.

## Quickstart

```sh
curl -fsSL https://raw.githubusercontent.com/EOEboh/hookdrop/main/scripts/install.sh | sh
hookdrop login                       # opens your browser to authorize
hookdrop listen my-slug -f 3000      # stream webhooks + forward to localhost:3000
```

## Install

### curl (recommended — one line, no extra steps)

```sh
curl -fsSL https://raw.githubusercontent.com/EOEboh/hookdrop/main/scripts/install.sh | sh
```

Downloads the latest release for your OS/arch, verifies it against
`checksums.txt`, and installs to `/usr/local/bin` (or `~/.local/bin`).

### Homebrew (macOS / Linux)

Homebrew requires you to **trust a third-party tap** before it will install
from it, so this is a three-step, one-time setup:

```sh
brew tap EOEboh/hookdrop
brew trust eoeboh/hookdrop      # one-time: approve this tap
brew install hookdrop
```

Upgrade later with `brew upgrade hookdrop`. (Without `brew trust`, the install
fails with "Refusing to load formula … from untrusted tap".)

### Windows

Download the archive from [GitHub Releases](https://github.com/EOEboh/hookdrop/releases).
(Note: replacing `hookdrop.exe` while a `listen` session is running will fail
on Windows — stop it first. On macOS/Linux, upgrading while running is fine;
the active session keeps the old binary until it exits.)

## Use

```sh
hookdrop login                       # browser-based; --token or --no-browser for headless
hookdrop whoami                      # account, plan, limits
hookdrop endpoints                   # list your named endpoints

hookdrop listen                      # pick an endpoint, stream webhooks
hookdrop listen my-slug              # stream a specific endpoint
hookdrop listen my-slug -f 3000      # …and forward each one to localhost:3000
hookdrop listen my-slug -f 3000 --path /webhook   # forward to a specific path
hookdrop listen my-slug -f https://localhost:8443/hook  # full URL also works

hookdrop logout
```

`listen` takes the endpoint as an optional argument: with one named endpoint
it's chosen automatically, with several you're prompted to pick. `-f/--forward`
accepts a bare port (`3000`), `host:port`, or a full URL; `--path` appends a
path. Without `-f`, it just streams webhooks to your terminal.

Shell completions are available via `hookdrop completion bash|zsh|fish|powershell`.

Forwarded requests carry `X-Hookdrop-Forwarded: true` and
`X-Hookdrop-Original-Id` headers; hop-by-hop headers (`Host`,
`Content-Length`, …) are regenerated, everything else — including provider
signature headers — is preserved byte-for-byte.

## Config

Stored at the OS config dir (`~/.config/hookdrop/config.json` on Linux,
`~/Library/Application Support/hookdrop/config.json` on macOS,
`%AppData%\hookdrop\config.json` on Windows), file mode 0600.

Environment overrides: `HOOKDROP_TOKEN`, `HOOKDROP_API_URL`,
`HOOKDROP_FRONTEND_URL`. `NO_COLOR` disables ANSI colors.

## Development

```sh
cd cli
CGO_ENABLED=0 go build -o hookdrop .
HOOKDROP_API_URL=http://localhost:8080 ./hookdrop login
```

Releases are run locally with goreleaser (no GitHub Actions required).
Tag the release, then run goreleaser from `cli/` with two free personal
access tokens in the environment:

```sh
git tag v0.1.0
git push origin v0.1.0

cd cli
export GITHUB_TOKEN=<PAT with Contents:write on EOEboh/hookdrop>
export HOMEBREW_TAP_GITHUB_TOKEN=<fine-grained PAT on EOEboh/homebrew-hookdrop>
go run github.com/goreleaser/goreleaser/v2@latest release --clean
```

This publishes the GitHub Release for darwin/linux (amd64+arm64) and
windows/amd64, uploads `checksums.txt`, and updates the Homebrew tap
`EOEboh/homebrew-hookdrop`. `--snapshot --skip=publish` does a dry run.
