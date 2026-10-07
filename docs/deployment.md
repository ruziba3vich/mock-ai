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
- uses `GEMINI_CONTENT_MODEL` (default `gemini-3.7-flash`) for asynchronous admin PDF extraction when the encrypted Gemini key vault is configured;
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
## Gemini Writing evaluation

Migration 000010 adds encrypted provider keys, owned writing submissions and a
durable evaluation queue. `APP_AI_KEY_ENCRYPTION_KEY` is a GitHub Actions secret
containing 32 random bytes encoded as 64 hex characters; CI passes it as
`AI_KEY_ENCRYPTION_KEY`. Back up this secret securely: changing or losing it makes
stored keys unreadable until they are re-encrypted/reimported. Never commit it.

`GEMINI_WRITING_MODEL` is an optional Actions variable, defaulting to the verified
stable `gemini-3.7-flash`. There is no silent model-family fallback. The actual
provider-reported model version is stored in each report. HTTP calls are bounded
to 95 seconds. The API worker runs independently of student HTTP request lifetimes.

The operator-only `/app/mock-ai-ai-keys` binary reads a JSON array of Gemini API
keys from stdin and stores AES-256-GCM ciphertext with provider-specific associated
data. Fingerprints deduplicate imports; existing ciphertext is not overwritten.
Import only the authorized source entries named `gemini-api-key`. Do not import
the U-Code access bearer or unrelated provider credentials. The import command
never prints key values. No frontend or API route exports stored keys.

The current worker uses one enabled primary key, ordered by creation time and ID.
Other imported keys are retained for operator-managed rotation. It does not cycle
keys to evade a provider rate limit. Provider failures produce a generic safe
error; no raw provider body, essay, credential or authorization header is logged.
If the master secret or keys are absent, evaluation is unavailable but drafts work.

Local verification: regular `go test ./...`; isolated database checks use the
existing `MOCKAI_INTEGRATION=1` suite. The optional `TestLiveWriting` test accepts
`MOCKAI_LIVE_GEMINI_KEY` and sends only a checked-in synthetic essay. Do not enable
live calls in routine CI. Model scoring still needs human calibration before it
is used for consequential assessments.

Speaking reuses the same encrypted key vault. Optional Actions variables
`GEMINI_SPEAKING_MODEL`, `GEMINI_SPEAKING_VOICE_MODEL`, and `GEMINI_SPEAKING_LIVE_MODEL` select evaluation, legacy TTS,
and live conversation models; defaults are `gemini-3.7-flash`, `gemini-3.8-flash-tts`, and `gemini-3.8-live`. Migrations 11–12 are
applied by the normal backend deployment. Never rotate the encryption master
without re-encrypting both provider credentials and Speaking recordings.
See [Speaking operations and tests](speaking.md).
