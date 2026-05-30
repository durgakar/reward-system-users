.PHONY: run test build tidy dry-run docker-mail

run:
	go run ./cmd/rewards -config config/config.yaml

dry-run:
	go run ./cmd/rewards -config config/config.example.yaml

test:
	go test ./...

build:
	go build -o bin/rewards ./cmd/rewards

tidy:
	go mod tidy

docker-mail:
	docker compose up -d mailhog
