# Subdux

[![CI](https://github.com/kasuha07/subdux/actions/workflows/ci.yml/badge.svg)](https://github.com/kasuha07/subdux/actions/workflows/ci.yml)
[![GHCR](https://img.shields.io/badge/GHCR-ghcr.io%2Fkasuha07%2Fsubdux-2ea44f?logo=docker)](https://github.com/kasuha07/subdux/pkgs/container/subdux)
[![License: GPL-3.0-or-later](https://img.shields.io/badge/License-GPL--3.0--or--later-blue.svg)](LICENSE)

**Language:** English | [简体中文](README.zh-CN.md)

**Subdux** is a self-hosted subscription tracker for recurring bills, renewals, and reminders.
It combines a Go backend and a React frontend into a **single deployable binary** with an embedded SPA, while still supporting container-based deployment for homelabs and production servers.

Track SaaS tools, domains, streaming services, cloud servers, developer subscriptions, or any recurring expense — then get notified before renewal.

## Highlights

- **Built for self-hosting** — run it as a single binary or a container.
- **Purpose-built for subscriptions** — recurring subscriptions, dashboard summaries, categories, payment methods, icons, and calendar views.
- **Multi-currency ready** — record subscriptions in different currencies and aggregate totals into a preferred currency.
- **Reminder system included** — reminder policies, templates, previews, test sends, logs, and multiple notification channels.
- **Modern authentication** — email/password, password reset, TOTP 2FA, passkeys/WebAuthn, OIDC, and scoped API keys.
- **Admin-ready** — user management, registration controls, SMTP/OIDC settings, exchange-rate management, stats, and backup/restore.
- **Portable data** — native JSON export/import, Wallos import, and tokenized calendar feeds.

## Feature Overview

| Area | Included capabilities |
| --- | --- |
| Subscription tracking | Recurring subscriptions, next billing dates, notes, categories, payment methods, icons, dashboard summary |
| Notifications | Renewal reminders, day-based reminder policy, templates, previews, test sends, delivery logs |
| Notification channels | SMTP, Resend, Telegram, Webhook, Gotify, ntfy, Bark, ServerChan3, PushDeer, pushplus, Pushover, Feishu, WeCom, DingTalk, NapCat |
| Authentication | Email/password, forgot/reset password, TOTP + backup codes, passkeys/WebAuthn, OIDC, API keys |
| Administration | User management, registration controls, email domain whitelist, SMTP settings, OIDC settings, stats, backup/restore |
| Import / export | Native Subdux export/import, Wallos import, calendar feed tokens, API access |
| Localization | English, Simplified Chinese (`zh-CN`), Japanese (`ja`) |

## Quick Start

### Option 1: Run the published container image

Replace `<version>` with a release tag such as `0.8.1`.

```bash
docker run -d \
  --name subdux \
  -p 8080:8080 \
  -e DATA_PATH=/data \
  -e JWT_SECRET=replace-with-a-long-random-string \
  -v subdux-data:/data \
  ghcr.io/kasuha07/subdux:<version>
```

Then open <http://localhost:8080>.

> On a fresh instance, **the first registered user becomes the admin user**.

### Option 2: Use the bundled Docker Compose file

The repository includes a `docker-compose.yml` that builds the image locally.

```bash
docker compose up --build -d
```

This starts Subdux on port `8080` and stores persistent data in the `subdux-data` volume.

## Configuration

### Key environment variables

| Variable | Default | Notes |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listen port |
| `DATA_PATH` | `data` | Directory for the SQLite database, uploaded assets, and generated local keys |
| `JWT_SECRET` | auto-generated on first run if unset | Recommended in production; must be at least 32 characters |
| `SETTINGS_ENCRYPTION_KEY` | falls back to `JWT_SECRET`, then a generated local key file | Used to encrypt sensitive system settings and notification secrets |
| `ACCESS_TOKEN_TTL_MINUTES` | `15` | Access token lifetime |
| `REFRESH_TOKEN_TTL_HOURS` | `720` | Refresh token lifetime |
| `CORS_ALLOW_ORIGINS` | unset | Comma-separated list of allowed origins |
| `TZ` | system timezone | IANA timezone such as `UTC` or `Asia/Shanghai` |

### Production notes

- Mount `DATA_PATH` to persistent storage.
- Set stable `JWT_SECRET` and `SETTINGS_ENCRYPTION_KEY` values in production.
- Configure `CORS_ALLOW_ORIGINS` and/or the in-app `site_url` when serving behind a public domain.
- Configure SMTP before enabling email verification, password reset, or email notifications.
- If you use OIDC, make sure the redirect URL configured in Subdux exactly matches the provider configuration.
- Passkeys and OIDC generally require correct public URL and HTTPS configuration.

## Architecture

Subdux uses a monorepo, but deploys as a single application:

- **Backend:** Go 1.25 + Echo + GORM + SQLite
- **Frontend:** React 19 + Vite + TypeScript + Tailwind CSS v4 + Shadcn/UI
- **Deployment model:** the frontend is built into `web/dist`, then embedded into the Go binary via `go:embed`

Runtime routing:

- `/` — SPA frontend
- `/api/*` — REST API
- `/uploads/*` — uploaded assets
- `/api/calendar/feed` — tokenized read-only calendar feed

Background jobs started by the server include:

- exchange-rate refresh
- pending notification processing

## Project Structure

```text
subdux/
├── cmd/server/          # Go server entrypoint
├── internal/            # Backend application code
│   ├── api/             # Echo handlers and routing
│   ├── model/           # GORM models
│   ├── pkg/             # Shared infrastructure helpers
│   └── service/         # Business logic
├── web/                 # React frontend
├── frontend.go          # go:embed for web/dist
├── Makefile             # Common build commands
├── Dockerfile           # Multi-stage container build
└── docker-compose.yml
```

## Development

### Requirements

- Go **1.25+**
- Bun **1.x**
- Optional: `tmux` for `make dev`

### Local development

Because the frontend is embedded into the backend binary, you need `web/dist` before running the Go server directly.

```bash
# Build frontend assets for embedding
make frontend

# Run backend
go run ./cmd/server
```

For frontend development in a second terminal:

```bash
cd web
bun dev
```

Default development URLs:

- Backend: <http://localhost:8080>
- Frontend dev server: <http://localhost:5173>
- `/api` requests are proxied from Vite to the backend

### Make targets

```bash
make frontend   # bun install + production frontend build
make build      # frontend build + Go binary build
make dev        # tmux session running backend + Vite
make docker     # local Docker image build
make clean      # remove local binary
```

### Checks

```bash
go test ./...

cd web
bun run lint
bun run build
```

## Releases and Distribution

- CI runs on pushes and pull requests to `main`.
- Version tags like `v0.8.1` publish multi-architecture container images to GHCR.
- Container image: `ghcr.io/kasuha07/subdux`
- Releases: <https://github.com/kasuha07/subdux/releases>

## Contributing

Issues and pull requests are welcome.

Before opening a PR, please run:

```bash
go test ./...
cd web && bun run lint && bun run build
```

If you are contributing UI or frontend behavior, keep changes inside the feature-folder structure under `web/src/features/` and avoid editing generated files under `web/src/components/ui/`.

## License

Subdux is licensed under **GPL-3.0-or-later**. See [`LICENSE`](LICENSE).
