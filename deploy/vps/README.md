# VPS development infrastructure

This stack runs the shared Mock AI development services on an isolated Docker network. PostgreSQL, Redis, and MinIO do not publish host ports. A WireGuard peer reaches them through the private network.

| Service | VPN address | Port |
| --- | --- | --- |
| PostgreSQL | `172.30.50.10` | `5432` |
| Redis | `172.30.50.11` | `6379` |
| MinIO API | `172.30.50.12` | `9000` |
| MinIO console | `172.30.50.12` | `9001` |

The deployed copy lives at `/home/statistics/tools/mock-ai`. Secrets are held in its mode-`0600` `.env` file and WireGuard private keys are held under `vpn/config`; neither belongs in Git.

To start the stack on the VPS:

```sh
cd /home/statistics/tools/mock-ai
docker network inspect mock_ai_private >/dev/null 2>&1 || \
  docker network create --subnet 172.30.50.0/24 mock_ai_private
docker compose --env-file .env -f vpn/docker-compose.yaml up -d
docker volume inspect mock_ai_postgres_data >/dev/null 2>&1 || \
  docker volume create mock_ai_postgres_data
docker compose --env-file .env -f postgres/docker-compose.yaml up -d
docker compose --env-file .env -f redis/docker-compose.yaml up -d
docker compose --env-file .env -f minio/docker-compose.yaml up -d
```

The laptop peer config is generated at `vpn/config/peer_macbook/peer_macbook.conf`.

PostgreSQL explicitly uses the `mock-ai-postgres` Compose project and the external
`mock_ai_postgres_data` volume. Keep this project name unique: directory-derived
names such as `postgres` can collide with unrelated stacks. Do not override it
with a generic `-p` or `COMPOSE_PROJECT_NAME`, and never run `down` against the old
generic project to migrate it. The database publishes no host port; its private
`172.30.50.10:5432` address does not occupy the VPS host's port 5432.
