#!/bin/sh
set -eu

: "${IMAGE_REPOSITORY:?IMAGE_REPOSITORY is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"

deploy_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$deploy_root"

compose() {
  docker compose --env-file api/.env -f api/docker-compose.yaml "$@"
}

compose pull api migrate
compose --profile tools run --rm migrate
compose up -d --no-deps api

attempt=0
while [ "$attempt" -lt 18 ]; do
  status=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' mock_ai_api 2>/dev/null || true)
  if [ "$status" = "healthy" ]; then
    docker image prune -f --filter "until=168h" >/dev/null
    printf 'deployed %s:%s\n' "$IMAGE_REPOSITORY" "$IMAGE_TAG"
    exit 0
  fi
  if [ "$status" = "unhealthy" ] || [ "$status" = "exited" ] || [ "$status" = "dead" ]; then
    docker logs --tail 100 mock_ai_api >&2 || true
    exit 1
  fi
  attempt=$((attempt + 1))
  sleep 5
done

docker logs --tail 100 mock_ai_api >&2 || true
printf 'deployment timed out waiting for a healthy API\n' >&2
exit 1
