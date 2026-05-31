.PHONY: run serve migrate test build tidy dry-run docker-up docker-down integration

BINARY=reward-system-users
CMD=./cmd/reward-system-users

run:
	go run $(CMD) run -config config/config.yaml

serve:
	go run $(CMD) serve -config config/config.yaml

migrate:
	go run $(CMD) migrate -config config/config.yaml

dry-run:
	go run $(CMD) run -config config/config.example.yaml

test:
	go test ./... -count=1 -race

integration:
	go test ./internal/store/... -count=1 -tags=integration

build:
	go build -o bin/$(BINARY) $(CMD)

tidy:
	go mod tidy

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-mail:
	docker compose up -d mailhog
