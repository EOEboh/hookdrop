# Releasing the hookdrop CLI

Releases run **locally** with [goreleaser](https://goreleaser.com) — there is
no GitHub Actions involved. One command tags a version and publishes the
binaries and the Homebrew formula.

## Mental model: commits vs releases

- **Commits / merges** are just git. Push and merge as normal — nothing else
  happens, no tokens, no tags.
- **A release** is a deliberate, occasional event: it builds the CLI for all
  platforms, creates a GitHub Release, and updates the Homebrew tap so users
  can `brew upgrade`. You do this **once per version**, whenever you've
  accumulated changes worth shipping — not on every commit.

## One-time setup

1. **Create a fine-grained GitHub token** (Settings → Developer settings →
   Fine-grained tokens) with:
   - **Repository access → Only select repositories →** both
     `EOEboh/hookdrop` **and** `EOEboh/homebrew-hookdrop`.
   - **Permissions → Contents: Read and write.**

   The token value is shown only once at creation/regeneration — copy it then.

2. **Store it** where the release script can find it, in `~/.hookdrop-release.env`
   (outside the repo, so it can never be committed):

   ```sh
   echo 'export GITHUB_TOKEN=github_pat_your_token_here' > ~/.hookdrop-release.env
   chmod 600 ~/.hookdrop-release.env
   ```

   One token with access to both repos covers everything — the script reuses
   `GITHUB_TOKEN` as `HOMEBREW_TAP_GITHUB_TOKEN` automatically. (If you prefer
   two separate tokens, also set `export HOMEBREW_TAP_GITHUB_TOKEN=…` in the
   file.)

## Cutting a release

From `main`, up to date and clean:

```sh
git checkout main && git pull
scripts/release.sh v0.1.1
```

That's it. The script:

1. Checks you're on `main`, the tree is clean, and you're up to date.
2. Loads the token and verifies it can reach both repos (fails early if not).
3. Creates and pushes the `v0.1.1` tag.
4. Runs goreleaser: builds darwin/linux (amd64+arm64) and windows/amd64,
   writes `checksums.txt`, creates the GitHub Release, and pushes the updated
   `Formula/hookdrop.rb` to `EOEboh/homebrew-hookdrop`.

**Dry run** (build artifacts into `cli/dist/`, tag and publish nothing):

```sh
scripts/release.sh v0.1.1 --dry-run
```

## Verify a release

```sh
brew update && brew upgrade hookdrop      # existing users
brew install EOEboh/hookdrop/hookdrop     # brand-new user
hookdrop --version                        # prints the tag
```

Also confirm the GitHub Release page lists 5 archives + `checksums.txt`, and
the tap repo's `Formula/hookdrop.rb` points at the new version.

## Troubleshooting

- **`token cannot access …/… (HTTP 404)`** — the token doesn't have that repo
  selected, or lacks Contents: Read and write. Edit the token (see setup) and,
  if you regenerate it, update `~/.hookdrop-release.env` with the new value.
- **goreleaser failed after the tag was pushed** — the tag exists remotely but
  the release didn't finish. Delete it and rerun:
  ```sh
  git push --delete origin v0.1.1 && git tag -d v0.1.1
  scripts/release.sh v0.1.1
  ```
- **`working tree is dirty`** — commit or stash first; goreleaser refuses to
  release from a dirty tree.
- **Releasing from a non-main branch** (e.g. a hotfix) — prefix with
  `ALLOW_BRANCH=1 scripts/release.sh v0.1.1`.

## Version numbers

Plain `vMAJOR.MINOR.PATCH` tags (e.g. `v0.1.0`, `v0.2.0`, `v1.0.0`). These tag
CLI releases only — the backend deploys separately via `scripts/deploy.sh` and
is never tagged, so the version namespace is free for the CLI.
