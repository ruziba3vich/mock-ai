# Mock AI

An IELTS mock-exam platform for Telegram Mini Apps and desktop browsers. The first release covers Listening and Reading. Writing, Speaking, and the AI teacher are intentionally deferred.

## Repository layout

- `apps/api` — Go API, domain modules, migrations, and sqlc queries
- `apps/web` — responsive React client shared by Telegram and desktop
- `docs` — architecture and MVP decisions

## Local development

Requirements: Go 1.27+, Node.js 22+, Docker, and sqlc.

```sh
cp .env.example .env
docker compose up -d postgres
make sqlc
make api
```

In another terminal:

```sh
make web-install
make web
```

The API listens on `http://localhost:8080`; the web client listens on `http://localhost:5173`.

## Shared development services

The shared development PostgreSQL, Redis, and MinIO instances run on the VPS and are reachable over WireGuard. The local peer configuration is installed at `~/.config/wireguard/mock-ai.conf`.

```sh
make vpn-up
```

The generated local `.env` contains the private service addresses and credentials. Stop the tunnel with `make vpn-down`. VPS deployment definitions live in [deploy/vps](deploy/vps).

## MVP

The first vertical slice is: authenticate, browse published tests, begin a timed attempt, autosave answers, submit, calculate a Listening or Reading band score, and show attempt history. See [docs/mvp.md](docs/mvp.md).

Implemented API routes and payload conventions are documented in [docs/api.md](docs/api.md).

Backend CI/CD and the required GitHub secrets are documented in [docs/deployment.md](docs/deployment.md).
