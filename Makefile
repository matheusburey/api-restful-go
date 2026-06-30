-include .env

APP=gobid
BIN=bin/$(APP)

VERSION=$(shell git describe --tags --always)
COMMIT=$(shell git rev-parse --short HEAD)

.PHONY: help migrate sqlc run dev test clean tidy

help:
	@echo "Comandos:"
	@echo "  make migrate"
	@echo "  make sqlc"
	@echo "  make run"
	@echo "  make dev"
	@echo "  make test"
	@echo "  make tidy"
	@echo "  make clean"

migrate:
	github.com/matheusburey/api-restful-go

sqlc:
	sqlc generate -f ./internal/store/pgstore/sqlc.yml

run:
	go run ./cmd/api

dev:
	air --build.cmd "go build -o ./bin/api ./cmd/api" --build.entrypoint "./bin/api"

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin
	go clean