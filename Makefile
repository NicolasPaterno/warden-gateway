DB_URL=postgresql://postgres:warden@localhost:5432/warden?sslmode=disable

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ── Migrations ────────────────────────────────────────────────────────────────
migrate-up:
	migrate -path db/migration -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migration -database "$(DB_URL)" down

migrate-force:
	migrate -path db/migration -database "$(DB_URL)" force $(version)

# ── SQLC ──────────────────────────────────────────────────────────────────────
sqlc:
	sqlc generate

# ── Go ────────────────────────────────────────────────────────────────────────
build:
	go build -o bin/gateway ./cmd/gateway

run:
	go run ./cmd/gateway

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

tidy:
	go mod tidy

.PHONY: docker-up docker-down docker-logs migrate-up migrate-down migrate-force sqlc build run test test-coverage tidy
