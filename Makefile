.PHONY: run build test tidy db-up db-down migrate crawl

run:
	go run ./cmd/wend

build:
	go build -o bin/wend ./cmd/wend

test:
	go test ./...

tidy:
	go mod tidy

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

migrate:
	go run ./cmd/wend migrate

crawl:
	go run ./cmd/wend crawl bbc
