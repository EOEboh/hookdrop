#!/bin/bash
set -e

SERVER="deploy@178.104.166.5"
APP_DIR="/opt/hookdrop"

echo "→ Building Docker image..."
docker build --platform linux/amd64 -t hookdrop:latest .

echo "→ Saving image..."
docker save hookdrop:latest | gzip > /tmp/hookdrop.tar.gz

echo "→ Uploading to server..."
scp /tmp/hookdrop.tar.gz $SERVER:/tmp/hookdrop.tar.gz

echo "→ Loading and restarting on server..."
ssh $SERVER << 'EOF'
  docker load < /tmp/hookdrop.tar.gz
  cd /opt/hookdrop
  docker compose down
  docker compose up -d
  docker compose logs --tail=20
  rm /tmp/hookdrop.tar.gz
EOF

rm /tmp/hookdrop.tar.gz
echo "✓ Deployed successfully"