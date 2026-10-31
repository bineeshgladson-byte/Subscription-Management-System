# Common commands. Run `make help` to list them.
# Go does not need to be installed locally: Go commands run inside Docker.

COMPOSE := docker compose
GO_DOCKER := docker run --rm -v "$(CURDIR)/backend":/src -w /src -v brightloop-gomod:/go/pkg/mod -e GOTOOLCHAIN=local golang:1.26-alpine
-include .env
export

.PHONY: dev dev-backend dev-frontend dev-stop run setup wait-api \
        help up down restart logs ps migrate migrate-down seed psql reset mock-api \
        test-email be-test be-itest be-vet be-fmt be-tidy fe-get fe-run fe-test fe-analyze

# ---------------------------------------------------------------------------
# Day-to-day development (like the LMS project)
#   make dev           backend (hot reload) in the background + Flutter in front
#   make dev-backend   backend with hot reload, logs in this terminal
#   make dev-frontend  Flutter only (backend must be running)
#   make dev-stop      stop everything started by make dev
#   make run           backend from the built image (no hot reload) + Flutter
# ---------------------------------------------------------------------------
FE_PORT ?= 5000
FE_DEVICE ?= chrome
API_URL := http://localhost:$(or $(API_PORT),8080)

setup: ## First-time setup: create .env with a random JWT secret
	@if [ ! -f .env ]; then \
	  cp .env.example .env; \
	  sed -i "s|^JWT_SECRET=.*|JWT_SECRET=$$(openssl rand -hex 32)|" .env; \
	  echo "Created .env (random JWT_SECRET). Review it, then run: make dev"; \
	else echo ".env already exists, leaving it alone"; fi

dev: setup ## Run backend (hot reload) + Flutter web. Ctrl+C stops Flutter only
	@$(COMPOSE) stop api >/dev/null 2>&1 || true
	$(COMPOSE) --profile dev up -d --build db migrate mailpit swagger api-dev
	@$(MAKE) --no-print-directory wait-api
	@echo "Backend: $(API_URL)/api/v1  ·  logs: make logs-dev  ·  Mailpit: http://localhost:8025"
	@$(MAKE) --no-print-directory dev-frontend

dev-backend: setup ## Backend only with hot reload, logs in this terminal
	@$(COMPOSE) stop api >/dev/null 2>&1 || true
	$(COMPOSE) --profile dev up --build db migrate mailpit swagger api-dev

dev-frontend: ## Flutter web on :$(FE_PORT) against the running backend
	cd frontend && flutter pub get && \
	  flutter run -d $(FE_DEVICE) --web-port $(FE_PORT) --dart-define=API_BASE_URL=$(API_URL)/api/v1

dev-stop: ## Stop everything started by make dev (keeps data)
	$(COMPOSE) --profile dev down

logs-dev: ## Follow hot-reload API logs
	$(COMPOSE) --profile dev logs -f api-dev

run: setup ## Backend from the built image (no hot reload) + Flutter web
	@$(COMPOSE) --profile dev rm -sf api-dev >/dev/null 2>&1 || true
	$(COMPOSE) up -d --build
	@$(MAKE) --no-print-directory wait-api
	@$(MAKE) --no-print-directory dev-frontend

wait-api:
	@printf "Waiting for API"; for i in $$(seq 1 150); do \
	  curl -sf $(API_URL)/health >/dev/null && { echo " ✓ ready"; exit 0; }; printf "."; sleep 2; done; \
	  echo " ✗ API didn't start. See: docker compose --profile dev logs api-dev"; exit 1

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' Makefile | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start db, migrations, api, mailpit, swagger (no hot reload)
	@$(COMPOSE) --profile dev rm -sf api-dev >/dev/null 2>&1 || true
	$(COMPOSE) up -d --build

down: ## Stop everything (keeps data)
	$(COMPOSE) down

restart: ## Rebuild and restart only the API
	$(COMPOSE) up -d --build api

logs: ## Follow API logs
	$(COMPOSE) logs -f api

ps: ## Show running services
	$(COMPOSE) ps

migrate: ## Apply pending migrations
	$(COMPOSE) run --rm migrate

migrate-down: ## Roll back the LAST migration
	$(COMPOSE) run --rm migrate -path=/migrations -database "$(DATABASE_URL)" down 1

seed: ## Load demo catalogue (safe to repeat)
	@for f in database/seeds/*.sql; do echo "-> $$f"; \
	  $(COMPOSE) exec -T db psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB) -f /seeds/$$(basename $$f); done

psql: ## Open a psql shell
	$(COMPOSE) exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

reset: ## DANGER: delete the database volume and start fresh
	$(COMPOSE) down -v
	$(MAKE) up

test-email: ## Send one test email with the .env SMTP settings: make test-email TO=you@gmail.com
	@test -n "$(TO)" || (echo "usage: make test-email TO=you@example.com" && exit 2)
	$(COMPOSE) build api >/dev/null
	$(COMPOSE) run --rm --no-deps --entrypoint ./mailtest api "$(TO)"

mock-api: ## Start the Prism mock API on :4010 from openapi.yaml
	$(COMPOSE) --profile mock up -d mock

be-test: ## Run Go tests (in Docker)
	$(GO_DOCKER) go test ./...

be-itest: ## Integration tests against the docker-compose database (run `make up` first)
	docker run --rm --network brightloop_default -v "$(CURDIR)/backend":/src -w /src -v brightloop-gomod:/go/pkg/mod -e GOTOOLCHAIN=local \
	  -e TEST_DATABASE_URL="postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@db:5432/$(POSTGRES_DB)?sslmode=disable" \
	  golang:1.26-alpine go test -count=1 -v ./tests/integration/...

be-vet: ## go vet (in Docker)
	$(GO_DOCKER) go vet ./...

be-fmt: ## gofmt -w (in Docker)
	docker run --rm --user $$(id -u):$$(id -g) -v "$(CURDIR)/backend":/src -w /src golang:1.26-alpine gofmt -w .

be-tidy: ## go mod tidy (in Docker)
	$(GO_DOCKER) go mod tidy

fe-get: ## flutter pub get
	cd frontend && flutter pub get

fe-run: ## Run Flutter web against the local API
	cd frontend && flutter run -d chrome --web-port 5000 --dart-define=API_BASE_URL=http://localhost:$(API_PORT)/api/v1

fe-test: ## flutter test
	cd frontend && flutter test

fe-analyze: ## flutter analyze
	cd frontend && flutter analyze
