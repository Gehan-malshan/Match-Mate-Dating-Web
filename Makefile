.DEFAULT_GOAL := help

ACCOUNT_DIR := services/account-service
EVENT_DIR := services/event-service
MATCHMAKING_DIR := services/matchmaking-service
BOOKING_DIR := services/booking-service
PAYMENT_DIR := services/payment-service
NOTIFICATION_DIR := services/notification-service
MODERATION_DIR := services/moderation-service
GRAPHQL_DIR := services/graphql-gateway

.PHONY: help setup start backend apps frontend status logs stop test build

help: ## Show the basic local-development commands
	@echo "MatchMate — basic local commands"
	@echo ""
	@echo "  make setup       Install Bun and Go dependencies (first time only)"
	@echo "  make start       Start backend, then run the unified frontend in this terminal"
	@echo "  make backend     Start all current Docker backends and infrastructure"
	@echo "  make apps        Run the unified member/admin frontend"
	@echo "  make frontend    Run the unified frontend at http://localhost:5173"
	@echo "  make status      Show backend container status"
	@echo "  make logs        Follow backend logs"
	@echo "  make stop        Stop backend containers; keep database data"
	@echo "  make test        Run backend and frontend tests"
	@echo "  make build       Build Docker backend and frontend production files"

setup: ## Install project dependencies
	bun install --frozen-lockfile
	go -C $(ACCOUNT_DIR) mod download
	go -C $(EVENT_DIR) mod download
	go -C $(MATCHMAKING_DIR) mod download
	go -C $(BOOKING_DIR) mod download
	go -C $(PAYMENT_DIR) mod download
	go -C $(NOTIFICATION_DIR) mod download
	go -C $(MODERATION_DIR) mod download
	go -C $(GRAPHQL_DIR) mod download

start: backend ## Start the complete local project
	@echo "Backend is running. Starting the unified frontend now..."
	@echo "Open http://localhost:5173 after Vite prints its URLs."
	bun run dev

backend: ## Start/rebuild the current Docker backend stack
	docker compose up --build -d
	@echo "Account API: http://localhost:8081"
	@echo "RabbitMQ:    http://localhost:15672"
	@echo "Event API:   http://localhost:8082"
	@echo "Matching API:http://localhost:8083"
	@echo "Payment API: http://localhost:8084"
	@echo "Booking API: http://localhost:8085"
	@echo "Notify API:  http://localhost:8086"
	@echo "Moderation:  http://localhost:8087"
	@echo "GraphQL API: http://localhost:8080/graphql"

apps: ## Run the unified member and administration frontend
	bun run dev

frontend: ## Run the React/TanStack web app
	bun run dev:web

status: ## Show backend status
	docker compose ps -a

logs: ## Follow backend logs
	docker compose logs -f

stop: ## Stop containers but preserve PostgreSQL data
	docker compose down

test: ## Run implemented backend and frontend tests
	go -C $(ACCOUNT_DIR) test ./...
	go -C $(EVENT_DIR) test ./...
	go -C $(MATCHMAKING_DIR) test ./...
	go -C $(BOOKING_DIR) test ./...
	go -C $(PAYMENT_DIR) test ./...
	go -C $(NOTIFICATION_DIR) test ./...
	go -C $(MODERATION_DIR) test ./...
	go -C $(GRAPHQL_DIR) test ./...
	bun run test:web

build: ## Build the Docker backend and frontend production assets
	docker compose build
	go -C $(ACCOUNT_DIR) vet ./...
	go -C $(EVENT_DIR) vet ./...
	go -C $(MATCHMAKING_DIR) vet ./...
	go -C $(BOOKING_DIR) vet ./...
	go -C $(PAYMENT_DIR) vet ./...
	go -C $(NOTIFICATION_DIR) vet ./...
	go -C $(MODERATION_DIR) vet ./...
	go -C $(GRAPHQL_DIR) vet ./...
	bun run build:web
