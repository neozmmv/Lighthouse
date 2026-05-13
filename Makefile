LIGHTHOUSE_BIN ?= $(APPDATA)/lighthouse/bin
MINIO_DATA     ?= $(APPDATA)/lighthouse/data
MAKE := "C:/PROGRA~2/GnuWin32/bin/make.exe"

# IMPORTANT!
# THIS MAKEFILE IS RUN ON WINDOWS WITH GIT BASH
# ALSO, THIS WILL ONLY RUN IF YOU HAVE LIGHTHOUSE INSTALLED
# INSTALL LIGHTHOUSE FIRST FOR IT TO WORK, SINCE IT DEPENDS ON LOCAL BINARIES

.PHONY: all check build lint fmt typecheck \
        lint-frontend lint-backend lint-cli \
        fmt-backend fmt-cli \
        build-frontend build-backend build-cli \
        install dev dev-minio dev-frontend dev-backend dev-caddy \
        docker-up docker-down docker-build clean help

all: check build ## run all checks then build everything

# deps

install:
	cd frontend && npm install

# checks

check: lint typecheck

lint: lint-frontend lint-backend lint-cli

lint-frontend:
	cd frontend && npm run lint

lint-backend:
	cd backend-go && go vet ./...

lint-cli:
	cd cli && go vet ./...

typecheck:
	cd frontend && npx tsc --noEmit

fmt: fmt-backend fmt-cli

fmt-backend:
	cd backend-go && go fmt ./...

fmt-cli:
	cd cli && go fmt ./...

# builds

build: build-frontend build-backend build-cli

build-frontend:
	cd frontend && npm run build

build-backend:
	cd backend-go && go build ./...

build-cli:
	cd cli && go build ./...

# docker (dev)

docker-up:
	docker compose -f docker-compose.dev.yml up

docker-down:
	docker compose -f docker-compose.dev.yml down

docker-build:
	docker compose -f docker-compose.dev.yml build

# local dev

dev: ## start minio, backend, and frontend locally
	$(MAKE) -j4 dev-minio dev-caddy dev-backend dev-frontend

dev-minio:
	MINIO_ROOT_USER=lighthouse MINIO_ROOT_PASSWORD=lighthouse_secret \
	    "$(LIGHTHOUSE_BIN)/minio.exe" server "$(MINIO_DATA)" --console-address ":9001"

dev-frontend:
	cd frontend && npm run dev

dev-caddy:
	"$(LIGHTHOUSE_BIN)/caddy.exe" run --config Caddyfile.dev --watch

dev-backend:
	cd backend-go && go run .
clean:
	rm -rf frontend/dist
	cd backend-go && go clean ./...
	cd cli && go clean ./...


help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
