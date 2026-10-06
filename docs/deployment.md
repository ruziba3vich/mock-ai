# Backend deployment

The backend pipeline is defined in `.github/workflows/backend.yml`. Pull requests run Go tests and `go vet`. A push to `main` that changes backend or deployment files also builds the API image, publishes immutable `sha-<commit>` and `latest` tags to GHCR, runs database migrations, deploys that exact SHA to the VPS, and waits for Docker health checks.

## GitHub Actions secrets

Configure these repository or `production` environment secrets:

| Secret | Value |
| --- | --- |
| `VPS_HOST` | `46.8.176.85` |
| `VPS_USER` | `statistics` |
| `VPS_SSH_PRIVATE_KEY` | Private key whose public key is authorized for the VPS account |
| `VPS_KNOWN_HOSTS` | Trusted SSH host-key line for `46.8.176.85` |
| `GHCR_READ_TOKEN` | Fine-grained/classic token with permission to read the package; it may be empty only if the package is public or the VPS is already logged in to an authorized account |
| `APP_DATABASE_URL` | Production PostgreSQL URL using the private address, for example `postgres://...@172.30.50.10:5432/mock_ai?sslmode=disable` |
| `APP_SESSION_SECRET` | At least 32 cryptographically random characters |
| `APP_WEB_ORIGIN` | Public HTTPS origin of the web application |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token; enables Mini App authentication and the `/start` bot worker |
| `TELEGRAM_MINI_APP_URL` | Public HTTPS Mini App URL; optional and defaults to `APP_WEB_ORIGIN` |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID; may remain empty until Google is configured |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret; may remain empty until Google is configured |
| `GOOGLE_REDIRECT_URL` | Public API callback URL ending in `/api/v1/auth/google/callback` |

`GITHUB_TOKEN` publishes the image and does not need to be created manually. The workflow grants it `packages: write` only in the publishing job.

Use a trusted local `known_hosts` entry rather than collecting the host key inside CI:

```sh
ssh-keygen -F 46.8.176.85 -f ~/.ssh/known_hosts
```

## VPS behavior

Deployment files live at `/home/statistics/tools/mock-ai/api` and are updated by the workflow before each deployment. GitHub Actions constructs `api/.env`, uploads it as `.env.next`, assigns mode `0600`, and atomically renames it into place before running migrations. The infrastructure file at `/home/statistics/tools/mock-ai/.env` is not modified. The API:

- joins `mock_ai_private` to reach PostgreSQL;
- binds only to `127.0.0.1:8081` for a local Nginx reverse proxy;
- runs as an unprivileged container user;
- receives runtime configuration from GitHub Actions secrets through `/home/statistics/tools/mock-ai/api/.env`;
- runs embedded, versioned SQL migrations before replacing the API container;
- is considered deployed only after `/healthz` passes its container health check.

The public HTTPS domain and Nginx virtual host remain separate because no backend domain has been selected yet.
## Separate admin panel

`https://mockai.admin.chessinsight.uz` serves the administrator application,
not the student dashboard. Local development uses `/admin` on the Vite server.
`ADMIN_ORIGIN` defaults to that production origin, or `WEB_ORIGIN` in development.
Do not add an admin role to a student: it grants no publishing permissions.

Provision a new administrator from the trusted VPS shell after migration 000009.
The command reads the password on stdin, never from command arguments. There
are no seeded credentials, automatic promotions or public signup endpoints.
For example, in Bash (replace email/name):

```sh
read -r -s -p 'New admin password (12+ characters): ' mockai_admin_password
printf '%s' "$mockai_admin_password" | docker exec -i mock_ai_api /app/mock-ai-admin --email editor@example.com --name 'Content Editor' --password-stdin
unset mockai_admin_password
```

Passwords use the existing Argon2id implementation. Duplicate emails fail
rather than replacing an existing administrator. For operator-managed account
disabling, update the exact `admins.id` to `disabled=true`; all its sessions
are then denied. No student account or other project's database is involved.

Migration 000009 is additive and preserves existing content. Its downgrade
refuses to run while admin-authored content exists, to prevent attribution loss;
use a forward fix or an explicitly planned export/migration instead. Initial
admin provisioning is an operator step, not a GitHub Actions secret or build-time
password. Student and admin authentication are deliberately independent.
