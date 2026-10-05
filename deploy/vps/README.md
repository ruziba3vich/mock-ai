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
docker compose --env-file .env -f postgres/docker-compose.yaml up -d
docker compose --env-file .env -f redis/docker-compose.yaml up -d
docker compose --env-file .env -f minio/docker-compose.yaml up -d
```

The laptop peer config is generated at `vpn/config/peer_macbook/peer_macbook.conf`.
