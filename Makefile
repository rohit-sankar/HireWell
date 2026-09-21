.PHONY: help install dev-backend dev-frontend build test lint fmt migrate-up migrate-down seed docker-up docker-down

help:
	@echo "Available targets:"
	@echo "  install         Install backend + frontend dependencies"
	@echo "  dev-backend     Run the Go backend with hot-ish reload (go run)"
	@echo "  dev-frontend    Run the Next.js dev server"
	@echo "  build           Build backend binary + frontend production bundle"
	@echo "  test            Run backend Go tests (unit + integration)"
	@echo "  lint            Run backend go vet + frontend eslint"
	@echo "  fmt             gofmt the backend"
	@echo "  migrate-up      Apply pending DB migrations"
	@echo "  migrate-down    Roll back the most recent migration"
	@echo "  seed            Seed demo data (John Doe / Example Technologies)"
	@echo "  docker-up       Start the full stack via docker compose"
	@echo "  docker-down     Stop the docker compose stack"

install:
	cd backend && go mod download
	cd frontend && npm install

dev-backend:
	cd backend && go run ./cmd/server

dev-frontend:
	cd frontend && npm run dev

build:
	cd backend && go build -o ../bin/server ./cmd/server
	cd frontend && npm run build

test:
	cd backend && go test ./... -v

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

fmt:
	cd backend && gofmt -w .

migrate-up:
	cd backend && go run ./cmd/server -migrate-up

migrate-down:
	cd backend && go run ./cmd/server -migrate-down

seed:
	cd backend && go run ./seeds/seed.go

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
