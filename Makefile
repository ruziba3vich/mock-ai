.PHONY: api web web-install sqlc migrate-up test vpn-up vpn-down

api:
	set -a; . ./.env; set +a; cd apps/api && go run ./cmd/api

web:
	cd apps/web && npm run dev

web-install:
	cd apps/web && npm install

sqlc:
	cd apps/api && sqlc generate

migrate-up:
	docker compose run --rm migrate

test:
	cd apps/api && go test ./...
	cd apps/web && npm run build

vpn-up:
	sudo wg-quick up $(HOME)/.config/wireguard/mock-ai.conf

vpn-down:
	sudo wg-quick down $(HOME)/.config/wireguard/mock-ai.conf
