#!/bin/bash
# Turn a bare Ubuntu box into a hookdrop host.
#
# Written to be run once per server and re-run safely: every step checks
# before it acts. It hardcodes no IP and no provider, because the point of
# having it is that the *next* migration costs an afternoon instead of a week.
#
# Usage, from a machine that can already SSH to the box as its default user
# (ubuntu on Oracle and GCP images, root on most other VPS providers):
#
#   scp -r deploy scripts/provision.sh scripts/backup.sh ubuntu@<ip>:/tmp/
#   ssh ubuntu@<ip> 'sudo bash /tmp/provision.sh'
#
# Then, separately, because they carry secrets:
#   scp deploy/.env  deploy@<ip>:/opt/hookdrop/.env
#   ssh deploy@<ip> 'chmod 600 /opt/hookdrop/.env'
#
# What it does NOT do: open the cloud provider's own firewall. On Oracle that
# is a VCN security list or NSG rule and has to be done in the console or via
# the OCI CLI. See the note under "firewall" below — getting only one of the
# two layers right is the classic way to lose an afternoon here.

set -euo pipefail

APP_DIR="${APP_DIR:-/opt/hookdrop}"
APP_USER="${APP_USER:-deploy}"
SRC_DIR="${SRC_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"

# The public half of the key the Deploy workflow authenticates with
# (repo secret DEPLOY_SSH_KEY). Pass it in rather than baking it in:
#   DEPLOY_PUBKEY="ssh-ed25519 AAAA..." sudo -E bash provision.sh
DEPLOY_PUBKEY="${DEPLOY_PUBKEY:-}"

log() { printf '\n\033[1m→ %s\033[0m\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

[ "$(id -u)" -eq 0 ] || { echo "run as root (sudo)"; exit 1; }

# ── Packages ────────────────────────────────────────────────────────────────
log "Base packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
# awscli talks to R2 over the S3 API, for scripts/backup.sh.
apt-get install -y -qq ca-certificates curl gnupg sqlite3 rsync awscli \
	iptables-persistent netfilter-persistent

# ── Docker ──────────────────────────────────────────────────────────────────
if have docker; then
	log "Docker already installed ($(docker --version))"
else
	log "Installing Docker Engine"
	install -m 0755 -d /etc/apt/keyrings
	curl -fsSL https://download.docker.com/linux/ubuntu/gpg |
		gpg --dearmor -o /etc/apt/keyrings/docker.gpg
	chmod a+r /etc/apt/keyrings/docker.gpg
	echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
		>/etc/apt/sources.list.d/docker.list
	apt-get update -qq
	apt-get install -y -qq docker-ce docker-ce-cli containerd.io \
		docker-buildx-plugin docker-compose-plugin
	systemctl enable --now docker
fi

# ── Caddy ───────────────────────────────────────────────────────────────────
if have caddy; then
	log "Caddy already installed ($(caddy version | head -1))"
else
	log "Installing Caddy"
	curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key |
		gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
	curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
		>/etc/apt/sources.list.d/caddy-stable.list
	apt-get update -qq
	apt-get install -y -qq caddy
fi

# ── Application user and layout ─────────────────────────────────────────────
log "User '$APP_USER' and $APP_DIR"
if ! id -u "$APP_USER" >/dev/null 2>&1; then
	useradd --create-home --shell /bin/bash "$APP_USER"
fi
# Needs docker to run the deploy, and that is the whole of its privilege —
# no sudo. A deploy is a container restart, not root on the box.
usermod -aG docker "$APP_USER"

install -d -o "$APP_USER" -g "$APP_USER" -m 0755 "$APP_DIR" \
	"$APP_DIR/data" "$APP_DIR/backups"

if [ -n "$DEPLOY_PUBKEY" ]; then
	install -d -o "$APP_USER" -g "$APP_USER" -m 0700 "/home/$APP_USER/.ssh"
	touch "/home/$APP_USER/.ssh/authorized_keys"
	grep -qxF "$DEPLOY_PUBKEY" "/home/$APP_USER/.ssh/authorized_keys" ||
		echo "$DEPLOY_PUBKEY" >>"/home/$APP_USER/.ssh/authorized_keys"
	chown -R "$APP_USER:$APP_USER" "/home/$APP_USER/.ssh"
	chmod 600 "/home/$APP_USER/.ssh/authorized_keys"
else
	echo "  ! DEPLOY_PUBKEY unset — add the Deploy workflow's public key to"
	echo "    /home/$APP_USER/.ssh/authorized_keys before the first deploy."
fi

# ── Config ──────────────────────────────────────────────────────────────────
log "Installing compose and Caddyfile"
# Never clobber a live compose file: the Deploy workflow rewrites its image:
# line, so the copy on the box is ahead of the one in git by design.
if [ -f "$APP_DIR/docker-compose.yml" ]; then
	echo "  compose already present, left alone"
else
	install -o "$APP_USER" -g "$APP_USER" -m 0644 \
		"$SRC_DIR/../deploy/docker-compose.yml" "$APP_DIR/docker-compose.yml"
fi

install -m 0644 "$SRC_DIR/../deploy/Caddyfile" /etc/caddy/Caddyfile
install -d -o caddy -g caddy -m 0755 /var/log/caddy
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy 2>/dev/null || systemctl restart caddy

if [ ! -f "$APP_DIR/.env" ]; then
	install -o "$APP_USER" -g "$APP_USER" -m 0600 \
		"$SRC_DIR/../deploy/.env.example" "$APP_DIR/.env"
	echo "  ! $APP_DIR/.env created from the example — fill it in before starting."
fi

# ── Firewall ────────────────────────────────────────────────────────────────
# Two layers, and the second one is the trap. Oracle's Ubuntu images ship an
# INPUT chain that REJECTs everything except SSH, and it persists across
# reboots — so with the VCN security list wide open, 443 is still a black
# hole and nothing in the console explains why.
#
# Rules are INSERTed ahead of that REJECT rather than the chain being flushed:
# on a box whose only access is SSH, a flush is how you lock yourself out.
log "Firewall (host layer)"
for port in 80 443; do
	if iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null; then
		echo "  tcp/$port already allowed"
	else
		iptables -I INPUT 1 -p tcp --dport "$port" -m conntrack \
			--ctstate NEW,ESTABLISHED -j ACCEPT
		echo "  tcp/$port allowed"
	fi
done
netfilter-persistent save >/dev/null
echo "  ! Also open TCP 80 and 443 to 0.0.0.0/0 in the provider's own"
echo "    firewall (Oracle: VCN security list or NSG). Both layers, or neither works."

# ── Backups ─────────────────────────────────────────────────────────────────
log "Backup timer"
install -m 0755 "$SRC_DIR/backup.sh" "$APP_DIR/backup.sh"
chown "$APP_USER:$APP_USER" "$APP_DIR/backup.sh"

cat >/etc/systemd/system/hookdrop-backup.service <<UNIT
[Unit]
Description=Back up the hookdrop SQLite database to R2
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=$APP_USER
WorkingDirectory=$APP_DIR
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/backup.sh
UNIT

# A timer, not a cron installed by the deploy. Backups that ride along with
# deploys stop happening exactly when deploys stop — which is the quiet period
# right before you need one.
cat >/etc/systemd/system/hookdrop-backup.timer <<'UNIT'
[Unit]
Description=Daily hookdrop database backup

[Timer]
OnCalendar=daily
RandomizedDelaySec=30m
Persistent=true

[Install]
WantedBy=timers.target
UNIT

systemctl daemon-reload
systemctl enable --now hookdrop-backup.timer

# ── Done ────────────────────────────────────────────────────────────────────
log "Provisioned"
cat <<DONE

  $APP_DIR              app root, owned by $APP_USER
  $APP_DIR/data         SQLite database and WAL  ← the only durable state
  $APP_DIR/.env         secrets, 0600
  /etc/caddy/Caddyfile  TLS + reverse proxy

  Next:
    1. Open TCP 80/443 in the provider firewall (see above).
    2. Fill in $APP_DIR/.env.
    3. Restore the database into $APP_DIR/data/hookdrop.db.
    4. cd $APP_DIR && docker compose up -d
    5. curl -s https://<hostname>/health
    6. systemctl start hookdrop-backup.service   # prove backups work now,
                                                 # not when you need one

DONE
