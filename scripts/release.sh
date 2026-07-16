#!/usr/bin/env bash
# Release the hookdrop CLI: tag the version and publish binaries + the
# Homebrew formula with goreleaser. Runs entirely locally — no GitHub Actions.
#
# Usage:
#   scripts/release.sh v0.1.1
#   scripts/release.sh v0.1.1 --dry-run   # build artifacts, publish nothing
#
# Token: reads GITHUB_TOKEN (and optional HOMEBREW_TAP_GITHUB_TOKEN) from the
# environment, or sources them from ~/.hookdrop-release.env if that file
# exists (override the path with HOOKDROP_RELEASE_ENV). See RELEASING.md.
set -euo pipefail

VERSION="${1:-}"
DRY_RUN=0
[ "${2:-}" = "--dry-run" ] && DRY_RUN=1

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

die() { echo "release: $*" >&2; exit 1; }

# ── Validate version argument ────────────────────────────────────────────
[ -n "$VERSION" ] || die "usage: scripts/release.sh vX.Y.Z [--dry-run]"
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) die "version must look like v1.2.3 (got '$VERSION')" ;;
esac

# ── Preflight: branch, clean tree, up to date, tag free ──────────────────
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [ "$BRANCH" != "main" ] && [ "${ALLOW_BRANCH:-0}" != "1" ]; then
  die "on branch '$BRANCH', not main. Release from main, or set ALLOW_BRANCH=1 to override."
fi

git diff --quiet && git diff --cached --quiet || die "working tree is dirty — commit or stash first."

git fetch --quiet origin "$BRANCH"
if [ -n "$(git rev-list "HEAD..origin/$BRANCH" 2>/dev/null)" ]; then
  die "local $BRANCH is behind origin/$BRANCH — run 'git pull' first."
fi

if git rev-parse -q --verify "refs/tags/$VERSION" >/dev/null; then
  die "tag $VERSION already exists locally. Delete it (git tag -d $VERSION) or pick a new version."
fi

# ── Load release tokens ──────────────────────────────────────────────────
ENV_FILE="${HOOKDROP_RELEASE_ENV:-$HOME/.hookdrop-release.env}"
if [ -z "${GITHUB_TOKEN:-}" ] && [ -f "$ENV_FILE" ]; then
  echo "Loading tokens from $ENV_FILE"
  # shellcheck disable=SC1090
  . "$ENV_FILE"
fi
[ -n "${GITHUB_TOKEN:-}" ] || die "GITHUB_TOKEN is not set (export it or put it in $ENV_FILE). See RELEASING.md."
# One token can serve both repos if it has Contents:write on each.
export HOMEBREW_TAP_GITHUB_TOKEN="${HOMEBREW_TAP_GITHUB_TOKEN:-$GITHUB_TOKEN}"

# ── Verify token reaches both repos before we tag anything ───────────────
check_repo() {
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer $1" "https://api.github.com/repos/$2")"
  [ "$code" = "200" ] || die "token cannot access $2 (HTTP $code). Fix the token's repository access. See RELEASING.md."
}
check_repo "$GITHUB_TOKEN" "EOEboh/hookdrop"
check_repo "$HOMEBREW_TAP_GITHUB_TOKEN" "EOEboh/homebrew-hookdrop"

# ── goreleaser runner (installed binary, else go run) ────────────────────
if command -v goreleaser >/dev/null 2>&1; then
  GORELEASER=(goreleaser)
else
  GORELEASER=(go run github.com/goreleaser/goreleaser/v2@latest)
fi

# ── Dry run: build only, no tag, no publish ──────────────────────────────
if [ "$DRY_RUN" = "1" ]; then
  echo "Dry run for $VERSION (no tag, no publish)…"
  ( cd cli && "${GORELEASER[@]}" release --snapshot --clean --skip=publish )
  echo "Dry run complete — artifacts in cli/dist/. Nothing was tagged or published."
  exit 0
fi

# ── Tag, push, release ───────────────────────────────────────────────────
echo "Tagging $VERSION…"
git tag "$VERSION"
git push origin "$VERSION"

echo "Releasing $VERSION with goreleaser…"
if ( cd cli && "${GORELEASER[@]}" release --clean ); then
  echo "✓ Released $VERSION"
  echo "  Release:  https://github.com/EOEboh/hookdrop/releases/tag/$VERSION"
  echo "  Verify:   brew update && brew upgrade hookdrop   (or: brew install EOEboh/hookdrop/hookdrop)"
else
  echo "release: goreleaser failed AFTER the tag was pushed." >&2
  echo "  To retry cleanly, delete the tag and run again:" >&2
  echo "    git push --delete origin $VERSION && git tag -d $VERSION" >&2
  exit 1
fi
