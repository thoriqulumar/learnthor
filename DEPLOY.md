# Deploying Learnthor to an Ubuntu VPS (Tencent Cloud)

This guide covers two deployment paths on Ubuntu 20.04 / 22.04 / 24.04:

- **Path A — Docker Compose** (recommended, fastest to set up): one
  `docker compose up` brings up Postgres + Go backend + nginx. All services
  run in containers; no host-level Go/Node/Postgres install needed. See
  [§A](#path-a--docker-compose) below.
- **Path B — Bare metal**: PostgreSQL, nginx, and a systemd-managed Go
  binary installed directly on the host. Use this if you want the lowest
  possible memory overhead or your VPS doesn't run Docker comfortably.
  See [§B](#path-b--bare-metal) below.

Both paths serve over HTTP on the VPS public IP — a "TLS later" section at
the bottom shows where Let/Encrypt slots in once you point a domain at the
box.

---

## Path A — Docker Compose

### Architecture

```
            ┌────────────────────────────────────┐
            │  docker compose (single host)       │
            │                                    │
Internet ──▶│  frontend :80  (nginx)             │
            │    ├── /         → /usr/share/nginx/html  (Vite build)
            │    ├── /api/*    → backend:8080     │
            │    └── /health   → backend:8080     │
            │    │                               │
            │    ├── backend :8080  (Go, internal)│
            │    │     └── DATABASE_URL → db:5432  │
            │    │                               │
            │    └── db :5432  (Postgres, volume) │
            │          └── pgdata (named volume)  │
            └────────────────────────────────────┘
```

Only port 80 (and 22 for SSH) is exposed to the host. The backend and
Postgres are internal to the compose network.

### A.1 Prerequisites

- A Tencent Lighthouse / CVM instance, Ubuntu 22.04 LTS (≥ 2 vCPU / 2 GB
  RAM / 20 GB disk).
- SSH key access as a non-root sudo user.
- Tencent security group / firewall: allow inbound **TCP 22** and
  **TCP 80** from `0.0.0.0/0`.

### A.2 Install Docker + Compose

```bash
sudo apt update && sudo apt -y upgrade
sudo apt install -y ca-certificates curl git ufw

# Docker Engine + Compose plugin (official repo)
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker   # pick up the group in this shell
docker --version && docker compose version
```

### A.3 Configure environment

```bash
git clone <your-repo> /opt/learnthor   # or rsync from your dev machine
cd /opt/learnthor

cp .env.docker.example .env.docker
nano .env.docker
# ── set these ─────────────────────────────────────────────────────────
# POSTGRES_PASSWORD=  (strong password for the DB)
# DATABASE_URL=       (must match POSTGRES_USER / PASSWORD above, host=db)
# JWT_SECRET=         (generate: openssl rand -hex 32)
# ─────────────────────────────────────────────────────────────────────
```

### A.4 Build + start the stack

```bash
docker compose up -d --build
```

This builds the backend + frontend images, starts Postgres (with a
healthcheck), the Go backend, and the nginx frontend. First build takes
~3–5 minutes (Go module download + Vite build); subsequent builds use
Docker layer cache and are much faster.

```bash
docker compose ps                 # all three should be Up / healthy
curl http://127.0.0.1/health      # → {"status":"ok"}
```

Visit `http://<VPS-IP>/` — you should see the login/register page.

### A.5 Seed the database (one-shot)

The seeder is a separate compose service (profile `seed`) so it doesn't
start with the main stack. Run it manually:

```bash
# Idempotent upsert (safe to re-run, preserves existing data):
docker compose run --rm seed

# Or wipe curriculum + progress (preserves users) then re-seed:
docker compose run --rm seed --reset --confirm-reset
```

The `--rm` flag removes the container after it exits; the image is already
built so the run is fast.

### A.6 Firewall

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw enable
```

### A.7 Operations runbook

```bash
# View logs (live tail)
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f db

# Restart a single service
docker compose restart backend

# Pull code updates and rebuild
git pull && docker compose up -d --build

# Update curriculum without deleting learner progress
docker compose run --rm seed

# Promote a user to admin
docker compose exec db psql -U learnthor -d learnthor \
  -c "UPDATE users SET role = 'admin' WHERE email = 'you@example.com';"

# Stop everything (data survives in the pgdata volume)
docker compose down

# Stop AND wipe the database volume (nuclear option — destroys all data)
docker compose down -v
```

### A.8 Backup (PostgreSQL dump)

```bash
# One-off manual backup
docker compose exec db pg_dump -U learnthor learnthor | gzip > learnthor-$(date +%F).sql.gz

# Nightly cron — add to /etc/cron.daily/learnthor-pgdump:
#!/bin/sh
umask 077
cd /opt/learnthor
docker compose exec -T db pg_dump -U learnthor learnthor | gzip > /var/backups/learnthor-$(date +%F).sql.gz
find /var/backups -name 'learnthor-*.sql.gz' -mtime +14 -delete
```

### A.9 Midtrans (payments) configuration

The backend reads Midtrans keys from the environment (set in `.env.docker`
for Docker, or `/etc/learnthor.env` for bare metal):

```
MIDTRANS_SERVER_KEY=SB-Mid-server-xxxxxxxxxxxxxxxx
MIDTRANS_CLIENT_KEY=SB-Mid-client-xxxxxxxxxxxxxxxx
MIDTRANS_ENV=sandbox
PREMIUM_PRICE_IDR=99000
```

1. **Get keys**: sign up at https://dashboard.midtrans.com → Settings →
   Access Keys. Sandbox keys start with `SB-Mid-`; production keys start
   with `Mid-`.
2. **Webhook URL**: in the Midtrans dashboard → Settings → Configuration →
   Payment Notification URL, set:
   ```
   http://<your-vps-ip>/api/v1/payments/notification
   ```
   (HTTP-only is fine for sandbox. Midtrans requires HTTPS for production
   webhooks — set this up after A.10 TLS is configured.)
3. **Sandbox testing**: use the Midtrans sandbox simulator (QRIS, bank
   transfer, card) at https://sandbox.midtrans.com/snap/v2/redirection/{token}.
   No real money moves. Sandbox payments trigger the webhook just like
   production, so premium grants flow through the same code path.
4. **Go live**: set `MIDTRANS_ENV=production` and swap in production keys,
   restart the backend, ensure the webhook URL is HTTPS.

### A.10 TLS / HTTPS (when you add a domain)

Swap the frontend nginx config to listen on 443 with certs, or — simpler —
put a reverse proxy (Caddy) in front of the whole stack. The minimal
Caddyfile approach:

```Caddyfile
learnthor.example.com {
    reverse_proxy localhost:80
}
```

Run Caddy on the host (or as a container with `network_mode: host`) and it
auto-provisions Let's Encrypt certs. Open `443/tcp` in UFW + the Tencent
security group. The JWT-in-localStorage auth flow works unchanged over
HTTPS.

---

## Path B — Bare metal

### Architecture

```
            ┌───────────────┐
Internet ──▶│  nginx :80    │── /          ──▶ /var/www/learnthor  (Vite build)
            │               │── /api/*     ──▶ 127.0.0.1:8080       (Go backend)
            │               │── /health   ──▶ 127.0.0.1:8080
            └───────────────┘
                    │
            ┌───────▼───────┐
            │ PostgreSQL :5432  (localhost only)
            └───────────────┘
```

The Go server binds to `127.0.0.1:8080`; nginx is the only public port
besides 22. PostgreSQL listens on `127.0.0.1:5432` only.

## B.1. Prerequisites

- A Tencent Lighthouse / CVM instance, Ubuntu 22.04 LTS (any ≥ 2 vCPU /
  2 GB RAM / 20 GB disk works for a small deployment).
- SSH key access as a non-root sudo user (the guide uses `ubuntu` as the
  example user — replace with yours).
- The Tencent security group / firewall must allow inbound **TCP 22** and
  **TCP 80** from `0.0.0.0/0`.

## B.2. System packages

```bash
sudo apt update && sudo apt -y upgrade
sudo apt install -y build-essential ufw nginx postgresql postgresql-contrib \
    ca-certificates curl git
```

### Go 1.23+ (toolchain for the backend)

The Ubuntu 22.04 apt package is Go 1.18 — too old for some modules. Install
the official toolchain:

```bash
cd /tmp
curl -LO https://go.dev/dl/go1.23.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.10.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
echo 'export PATH=$PATH:$(go env GOPATH)/bin' | sudo tee -a /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version   # should print go1.23.10
```

### Node 20 LTS (for the frontend build)

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node -v && npm -v
```

## B.3. PostgreSQL setup

```bash
sudo -u postgres psql <<'SQL'
CREATE USER learnthor WITH PASSWORD 'CHANGE_ME_STRONG';
CREATE DATABASE learnthor OWNER learnthor;
GRANT ALL PRIVILEGES ON DATABASE learnthor TO learnthor;
SQL
```

Confirm `/etc/postgresql/*/main/postgresql.conf` has
`listen_addresses = 'localhost'` (the default). Confirm
`/etc/postgresql/*/main/pg_hba.conf` has IPv4 local lines using
`md5` or `scram-sha-256` so the Go server can authenticate.

```bash
sudo systemctl enable --now postgresql
```

## B.4. Backend build + environment

### 4.1 Pull the code

```bash
sudo mkdir -p /opt/learnthor && sudo chown ubuntu:ubuntu /opt/learnthor
cd /opt/learnthor
# clone or rsync your repo here so /opt/learnthor/backend and
# /opt/learnthor/frontend both exist.
```

### 4.2 Environment file

Create `/etc/learnthor.env` (root-readable only — secrets live here):

```bash
sudo tee /etc/learnthor.env >/dev/null <<'EOF'
PORT=8080
APP_ENV=production
DATABASE_URL=host=127.0.0.1 user=learnthor password=CHANGE_ME_STRONG dbname=learnthor port=5432 sslmode=disable TimeZone=UTC
JWT_SECRET=REPLACE_WITH_64_HEX_CHARS
EOF

# Generate the JWT secret:
openssl rand -hex 32
# paste the output into JWT_SECRET=... above

sudo chown root:learnthor /etc/learnthor.env   # create the group first:
sudo groupadd learnthor
sudo usermod -aG learnthor ubuntu
sudo chmod 640 /etc/learnthor.env
```

> The systemd unit below loads this file via `EnvironmentFile=`. Re-login or
> `newgrp learnthor` to pick up the group membership in your current shell.

### 4.3 Build the server binary + seeder

```bash
cd /opt/learnthor/backend
go build -o /opt/learnthor/learnthor-server .
go build -o /opt/learnthor/learnthor-seed ./cmd/seed
```

### 4.4 Run migrations + seed

The server auto-migrates on boot, but you should seed explicitly:

```bash
# First-time seed (idempotent — safe to re-run):
/opt/learnthor/learnthor-seed

# To wipe curriculum + progress and re-seed (preserves users):
/opt/learnthor/learnthor-seed --reset --confirm-reset
```

### 4.5 systemd unit

```bash
sudo tee /etc/systemd/system/learnthor.service >/dev/null <<'EOF'
[Unit]
Description=Learnthor Go backend
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=ubuntu
Group=learnthor
WorkingDirectory=/opt/learnthor
EnvironmentFile=/etc/learnthor.env
ExecStart=/opt/learnthor/learnthor-server
Restart=on-failure
RestartSec=3
StandardOutput=append:/var/log/learnthor/server.log
StandardError=append:/var/log/learnthor/server.log

[Install]
WantedBy=multi-user.target
EOF

sudo mkdir -p /var/log/learnthor && sudo chown ubuntu:ubuntu /var/log/learnthor
sudo systemctl daemon-reload
sudo systemctl enable --now learnthor
sudo systemctl status learnthor --no-pager
```

Smoke test: `curl http://127.0.0.1:8080/health` should return
`{"status":"ok"}`.

## B.5. Frontend build

The SPA builds to `dist/`. Set `VITE_API_BASE_URL` to a **relative** path so
the browser asks nginx for `/api/v1/...` and nginx proxies to the backend
(no CORS, no hardcoded host):

```bash
cd /opt/learnthor/frontend
npm ci
VITE_API_BASE_URL=/api/v1 npm run build
# Output: /opt/learnthor/frontend/dist/
sudo mkdir -p /var/www/learnthor
sudo rsync -a --delete dist/ /var/www/learnthor/
sudo chown -R www-data:www-data /var/www/learnthor
```

Re-run these three lines every time you ship a frontend change.

## B.6. nginx configuration

```bash
sudo tee /etc/nginx/sites-available/learnthor >/dev/null <<'EOF'
server {
    listen 80;
    listen [::]:80;
    server_name _;   # IP-only; swap for your domain when you have one

    # Static SPA
    root /var/www/learnthor;
    index index.html;

    # SPA fallback: serve index.html for any path that isn't a real file
    # (React Router owns client-side routing).
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API reverse proxy → Go backend
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check (unauthenticated, useful for monitoring)
    location = /health {
        proxy_pass http://127.0.0.1:8080/health;
    }

    # Gzip text responses
    gzip on;
    gzip_types text/css application/javascript application/json image/svg+xml;
    gzip_min_length 1024;
}
EOF

sudo ln -sf /etc/nginx/sites-available/learnthor /etc/nginx/sites-enabled/learnthor
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx
```

Visit `http://<VPS-IP>/` — you should see the login/register page.
Visit `http://<VPS-IP>/health` — should return `{"status":"ok"}`.

## B.7. Firewall

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp     # SSH
sudo ufw allow 80/tcp     # HTTP
sudo ufw enable
sudo ufw status verbose
```

PostgreSQL is bound to `127.0.0.1` only, so it needs no UFW rule.

## B.8. Runbook

### Re-seed the database

```bash
# Update content in place; preserve learner progress:
sudo systemctl stop learnthor
/opt/learnthor/learnthor-seed
sudo systemctl start learnthor
```

### Update the codebase

```bash
cd /opt/learnthor
git pull            # or rsync from your dev machine

# Backend
cd backend && go build -o /opt/learnthor/learnthor-server . && \
    go build -o /opt/learnthor/learnthor-seed ./cmd/seed
sudo systemctl restart learnthor

# Frontend
cd ../frontend && npm ci && VITE_API_BASE_URL=/api/v1 npm run build && \
    sudo rsync -a --delete dist/ /var/www/learnthor/
```

### Tail the logs

```bash
sudo journalctl -u learnthor -f                  # systemd stream
tail -f /var/log/learnthor/server.log            # raw file
sudo tail -f /var/log/nginx/access.log           # nginx
```

### Promote a user to admin / editor

```bash
sudo -u postgres psql -d learnthor -c \
  "UPDATE users SET role = 'admin' WHERE email = 'you@example.com';"
```

Re-login in the browser so the JWT picks up the new role.

## B.9. TLS / HTTPS (when you add a domain)

When you point a domain at the VPS:

1. Update the Tencent security group / DNS A record to your VPS IP.
2. In `/etc/nginx/sites-available/learnthor`, change `server_name _;` to
   `server_name learnthor.example.com;`.
3. Install Caddy (simplest) or certbot:

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d learnthor.example.com
```

certbot rewrites the nginx config to listen on 443 with the certs and sets
up auto-renewal. Open `443/tcp` in UFW and the Tencent security group:

```bash
sudo ufw allow 443/tcp
```

Also bump `JWT_SECRET` rotation and `Cookie`/`SameSite` if you start using
cookies — the current JWT-in-localStorage flow works unchanged over HTTPS.

## B.10. Backup

The only stateful component is PostgreSQL. A nightly dump is enough for a
small deployment:

```bash
# /etc/cron.daily/learnthor-pgdump
sudo tee /etc/cron.daily/learnthor-pgdump >/dev/null <<'EOF'
#!/bin/sh
umask 077
pg_dump -U postgres learnthor | gzip > /var/backups/learnthor-$(date +%F).sql.gz
find /var/backups -name 'learnthor-*.sql.gz' -mtime +14 -delete
EOF
sudo chmod +x /etc/cron.daily/learnthor-pgdump
sudo mkdir -p /var/backups
```

The `ubuntu` user needs `sudo -u postgres` rights or a `.pgpass` entry for
the cron job to run unattended.

## B.11. Troubleshooting

| Symptom | Check |
|---|---|
| `502 Bad Gateway` from nginx | `systemctl status learnthor` — backend down? |
| `connection refused` on `/api` | backend not listening on `127.0.0.1:8080`; check `PORT` env |
| Login returns `INTERNAL_ERROR` | `JWT_SECRET` empty → server panics on startup |
| Cards never appear in review queue | run `/opt/learnthor/learnthor-seed` then study a module — SRS rows are created on first `SubmitAnswer` |
| Frontend 404s on refresh | nginx `try_files ... /index.html` missing |
| `psql: FATAL: password auth failed` | password in `/etc/learnthor.env` doesn't match the one you set in §3 |

See [the seed importer guide](backend/cmd/seed/README.md) for the canonical JSON schema, offline validation, backups, and safe updates.
