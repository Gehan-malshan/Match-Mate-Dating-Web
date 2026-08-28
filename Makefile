.DEFAULT_GOAL := help

COMPOSE_FILE := infrastructure/compose/account-profile.compose.yml
ACCOUNT_DIR := services/account-service

.PHONY: help doctor setup up status logs web test test-backend test-frontend vet build migrate seed down reset-local

help: ## Show available project commands
	@echo "MatchMate development commands"
	@echo ""
	@echo "  make doctor         Check required tools"
	@echo "  make setup          Install Bun and Go dependencies"
	@echo "  make up             Build/start backend, database, RabbitMQ, migration and seed"
	@echo "  make web            Start the frontend development server (second terminal)"
	@echo "  make status         Show all Compose containers, including one-shot jobs"
	@echo "  make logs           Follow backend/infrastructure logs"
	@echo "  make migrate        Re-run the idempotent migration job"
	@echo "  make seed           Restore shared development accounts"
	@echo "  make test           Run backend and frontend tests"
	@echo "  make vet            Run Go static analysis and frontend type checking"
	@echo "  make build          Build backend containers and frontend production assets"
	@echo "  make down           Stop local containers but preserve database data"
	@echo "  make reset-local CONFIRM=YES  Delete the local database volume"

doctor: ## Check required local tools
	@go version
	@bun --version
	@docker --version
	@docker compose version

setup: ## Install repository dependencies
	bun install --frozen-lockfile
	go -C $(ACCOUNT_DIR) mod download

up: ## Start the complete local Account/Profile backend
	docker compose -f $(COMPOSE_FILE) up -d --build
	@echo "Backend: http://localhost:8081"
	@echo "RabbitMQ dashboard: http://localhost:15672"
	@echo "Run 'make web' in a second terminal for the frontend."

status: ## Show infrastructure and one-shot job status
	docker compose -f $(COMPOSE_FILE) ps -a

logs: ## Follow all backend and infrastructure logs
	docker compose -f $(COMPOSE_FILE) logs -f

web: ## Start the member frontend
	bun run dev:web

migrate: ## Run the idempotent Account database migration
	docker compose -f $(COMPOSE_FILE) run --rm account-migrate

seed: ## Restore hard-coded development users and revoke their old sessions
	docker compose -f $(COMPOSE_FILE) run --rm account-seed

test: test-backend test-frontend ## Run all currently implemented automated tests

test-backend: ## Run Go unit tests
	go -C $(ACCOUNT_DIR) test ./...

test-frontend: ## Run frontend component tests
	bun run test:web

vet: ## Run static/type analysis
	go -C $(ACCOUNT_DIR) vet ./...
	bun run typecheck:web

build: ## Validate Compose and build backend/frontend artifacts
	docker compose -f $(COMPOSE_FILE) config --quiet
	docker compose -f $(COMPOSE_FILE) build
	bun run build:web

down: ## Stop containers while preserving local database data
	docker compose -f $(COMPOSE_FILE) down

reset-local: ## Delete all local Account database data; requires CONFIRM=YES
ifeq ($(CONFIRM),YES)
	docker compose -f $(COMPOSE_FILE) down -v
	@echo "Local MatchMate Account database volume deleted. Run 'make up' to recreate it."
else
	@echo "Refusing to delete local data. Re-run with: make reset-local CONFIRM=YES"
endif
