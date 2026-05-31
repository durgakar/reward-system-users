.PHONY: run test build tidy dry-run docker-mail

BINARY=reward-system-users
CMD=./cmd/reward-system-users

run:
	go run $(CMD) -config config/config.yaml

dry-run:
	go run $(CMD) -config config/config.example.yaml

test:
	go test ./...

build:
	go build -o bin/$(BINARY) $(CMD)

tidy:
	go mod tidy

docker-mail:
	docker compose up -d mailhog
