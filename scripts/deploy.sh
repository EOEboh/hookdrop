#!/bin/bash
# Break-glass deploy, straight from this machine.
#
# The normal route is the Deploy workflow in GitHub Actions (Actions tab →
# Deploy → Run workflow), which builds, pushes to GHCR, releases behind a
# health check and rolls back on failure. Use this only when Actions or GHCR
# is unavailable: it needs Docker locally, cross-compiles to amd64, and ships
# a ~13MB image over your connection.
#
# It also repoints docker-compose.yml at the locally loaded image, because the
# workflow leaves it pointing at a ghcr.io tag.
set -euo pipefail

SERVER="deploy@178.104.166.5"

echo "⚠  Break-glass deploy — prefer the Deploy workflow in GitHub Actions."
echo "   Working tree: $(git rev-parse --short HEAD)$([ -n "$(git status --porcelain)" ] && echo ' (DIRTY — uncommitted changes will ship)')"
read -r -p "   Continue? [y/N] " reply
[ "$reply" = "y" ] || { echo "aborted"; exit 1; }

echo "→ Building image (linux/amd64)..."
docker build --platform linux/amd64 -t hookdrop:latest .

echo "→ Uploading..."
docker save hookdrop:latest | gzip | ssh "$SERVER" 'cat > /tmp/hookdrop.tar.gz'

echo "→ Releasing..."
ssh "$SERVER" 'bash -s' <<'REMOTE'
set -euo pipefail
cd /opt/hookdrop
docker load < /tmp/hookdrop.tar.gz

cp docker-compose.yml "docker-compose.yml.bak-$(date +%F-%H%M%S)"
docker compose down
# Clean shutdown has checkpointed the WAL, so this snapshot is consistent.
cp data/hookdrop.db "data/hookdrop.db.pre-deploy-$(date +%F-%H%M%S)"

# The workflow points this at a ghcr.io tag; send it back to the local image.
sed -i 's|^\(\s*image:\).*|\1 hookdrop:latest|' docker-compose.yml
docker compose up -d

for i in $(seq 1 20); do
  if curl -fsS -m 5 http://localhost:8080/health >/dev/null 2>&1; then
    echo "✓ healthy on attempt $i"
    rm -f /tmp/hookdrop.tar.gz
    exit 0
  fi
  sleep 3
done

echo "✗ unhealthy after 60s"
docker logs hookdrop 2>&1 | tail -30
exit 1
REMOTE

echo "✓ Deployed"
