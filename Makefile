# Common commands. Run `make help` to list them.
# Go does not need to be installed locally: Go commands run inside Docker.

COMPOSE := docker compose
GO_DOCKER := docker run --rm -v "$(CURDIR)/backend":/src -w /src -v brightloop-gomod:/go/pkg/mod -e GOTOOLCHAIN=local golang:1.26-alpine
-include .env
export

.PHONY: help up down restart logs ps migrate migrate-down seed psql reset mock-api \
        be-test be-vet be-fmt be-tidy fe-get fe-run fe-test fe-analyze

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start db, migrations, api, mailpit, swagger
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

mock-api: ## Start the Prism mock API on :4010 from openapi.yaml
	$(COMPOSE) --profile mock up -d mock

be-test: ## Run Go tests (in Docker)
	$(GO_DOCKER) go test ./...

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
