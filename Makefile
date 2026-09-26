SOURCE ?= all

.PHONY: run build test tidy db-up db-down migrate crawl cluster crawl-bbc crawl-guardian

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
	go run ./cmd/wend crawl $(SOURCE)

cluster:
	go run ./cmd/wend cluster

crawl-bbc:
	go run ./cmd/wend crawl bbc

crawl-guardian:
	go run ./cmd/wend crawl guardian
