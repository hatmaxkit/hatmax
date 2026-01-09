# Makefile for Ticked example (hatmax single-binary app)

# Binary name
BINARY=ticked
LOG_LINES?=100

# Database configuration
DB_HOST?=localhost
DB_PORT?=5432
DB_USER?=dev
DB_PASS?=dev
DB_NAME?=tickedhm

# Server configuration
SERVER_PORT?=:8080

# Phony targets
.PHONY: help build run stop logs logs-clear fresh test clean db-init db-drop db-reset db-psql sqlc

help:
	@echo "Ticked (hatmax) commands:"
	@echo ""
	@echo "  make build       - Build the ticked binary"
	@echo "  make run         - Build and start ticked"
	@echo "  make stop        - Stop the running ticked process"
	@echo "  make logs        - Stream logs (tail -f)"
	@echo "  make logs-clear  - Clear log file"
	@echo "  make fresh       - Full reset: stop, clear logs, db-reset, and run"
	@echo "  make clean       - Clean binary, logs, and PIDs"
	@echo ""
	@echo "Database commands:"
	@echo "  make db-init     - Create PostgreSQL database '$(DB_NAME)'"
	@echo "  make db-drop     - Drop PostgreSQL database '$(DB_NAME)'"
	@echo "  make db-reset    - Drop and recreate database"
	@echo "  make db-psql     - Open psql shell to $(DB_NAME)"
	@echo ""
	@echo "Development commands:"
	@echo "  make seed        - Create superadmin user (dev mode)"
	@echo "  make sqlc        - Regenerate sqlc code"
	@echo "  make test        - Run tests"
	@echo ""
	@echo "Configuration (override with env vars):"
	@echo "  DB_HOST=$(DB_HOST)  DB_PORT=$(DB_PORT)  DB_USER=$(DB_USER)"
	@echo "  DB_NAME=$(DB_NAME)  SERVER_PORT=$(SERVER_PORT)"

# Build the binary
build:
	@echo "Building $(BINARY)..."
	@go build -o $(BINARY) .
	@echo "$(BINARY) built successfully"

# Run the application
run: build
	@echo "Starting $(BINARY)..."
	@$(MAKE) stop 2>/dev/null || true
	@port=$$(echo $(SERVER_PORT) | sed 's/^://'); \
		if lsof -ti:$$port >/dev/null 2>&1; then \
			echo "Killing process on port $$port..."; \
			lsof -ti:$$port | xargs -r kill -9 2>/dev/null || true; \
			sleep 1; \
		fi
	@export TICKED_DATABASE_HOST=$(DB_HOST) && \
		export TICKED_DATABASE_PORT=$(DB_PORT) && \
		export TICKED_DATABASE_USER=$(DB_USER) && \
		export TICKED_DATABASE_PASSWORD=$(DB_PASS) && \
		export TICKED_DATABASE_DATABASE=$(DB_NAME) && \
		export TICKED_DATABASE_SSLMODE=disable && \
		export TICKED_SERVER_PORT=$(SERVER_PORT) && \
		export TICKED_LOG_LEVEL=info && \
		nohup ./$(BINARY) > $(BINARY).log 2>&1 & echo $$! > $(BINARY).pid
	@sleep 2
	@if [ -f $(BINARY).pid ] && ps -p $$(cat $(BINARY).pid) > /dev/null 2>&1; then \
		echo ""; \
		echo "$(BINARY) started successfully!"; \
		echo "  PID: $$(cat $(BINARY).pid)"; \
		echo "  URL: http://localhost$(SERVER_PORT)"; \
		echo "  Log: $(BINARY).log"; \
		echo ""; \
		echo "First user to sign up becomes admin."; \
		echo ""; \
		echo "View logs: make logs"; \
		echo "Stop: make stop"; \
	else \
		echo "Failed to start $(BINARY). Check $(BINARY).log for errors."; \
		cat $(BINARY).log 2>/dev/null || true; \
		exit 1; \
	fi

# Run in foreground (for development)
run-fg: build
	@echo "Starting $(BINARY) in foreground..."
	@export TICKED_DATABASE_HOST=$(DB_HOST) && \
		export TICKED_DATABASE_PORT=$(DB_PORT) && \
		export TICKED_DATABASE_USER=$(DB_USER) && \
		export TICKED_DATABASE_PASSWORD=$(DB_PASS) && \
		export TICKED_DATABASE_DATABASE=$(DB_NAME) && \
		export TICKED_DATABASE_SSLMODE=disable && \
		export TICKED_SERVER_PORT=$(SERVER_PORT) && \
		export TICKED_LOG_LEVEL=debug && \
		./$(BINARY)

# Stop the application
stop:
	@if [ -f $(BINARY).pid ]; then \
		pid=$$(cat $(BINARY).pid); \
		if ps -p "$$pid" > /dev/null 2>&1; then \
			echo "Stopping $(BINARY) (PID: $$pid)..."; \
			kill "$$pid" 2>/dev/null || true; \
			sleep 1; \
			if ps -p "$$pid" > /dev/null 2>&1; then \
				kill -9 "$$pid" 2>/dev/null || true; \
			fi; \
		fi; \
		rm -f $(BINARY).pid; \
		echo "$(BINARY) stopped"; \
	else \
		echo "$(BINARY) is not running (no PID file)"; \
	fi

# Stream logs
logs:
	@if [ -f $(BINARY).log ]; then \
		echo "Streaming $(BINARY).log (last $(LOG_LINES) lines)..."; \
		echo "Press Ctrl+C to stop"; \
		echo ""; \
		tail -n $(LOG_LINES) -f $(BINARY).log; \
	else \
		echo "No log file found. Start with 'make run'"; \
	fi

# Clear logs
logs-clear:
	@echo "Clearing logs..."
	@rm -f $(BINARY).log
	@echo "Logs cleared"

# Fresh start
fresh:
	@echo "Fresh start: resetting environment..."
	@$(MAKE) stop
	@$(MAKE) logs-clear
	@$(MAKE) db-reset
	@$(MAKE) run
	@sleep 2
	@echo ""
	@echo "Ready. First signup becomes superadmin."
	@echo "Or run 'make seed' for dev user."

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Clean everything
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY) $(BINARY).log $(BINARY).pid
	@echo "Cleanup complete"

# Database: create
db-init:
	@echo "Creating PostgreSQL database '$(DB_NAME)'..."
	@command -v psql >/dev/null 2>&1 || { echo "psql not found. Install PostgreSQL client."; exit 1; }
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "CREATE DATABASE $(DB_NAME) OWNER $(DB_USER);" 2>/dev/null && \
		echo "Database '$(DB_NAME)' created" || \
		echo "Database '$(DB_NAME)' already exists"
	@echo "Ensuring public schema exists..."
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "CREATE SCHEMA IF NOT EXISTS public; GRANT ALL ON SCHEMA public TO $(DB_USER);" 2>/dev/null
	@echo "Database ready"

# Database: drop
db-drop:
	@echo "Dropping PostgreSQL database '$(DB_NAME)'..."
	@command -v psql >/dev/null 2>&1 || { echo "psql not found. Install PostgreSQL client."; exit 1; }
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);" 2>/dev/null
	@echo "Database '$(DB_NAME)' dropped"

# Database: reset
db-reset: db-drop db-init
	@echo "Database reset complete"

# Database: open psql shell
db-psql:
	@echo "Connecting to $(DB_NAME)..."
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME)

# NOTE: Seed is for dev convenience during frequent db resets.
# In production, first signup becomes superadmin automatically.
# This target may be removed in the future.
SEED_EMAIL?=superadmin@system.local
SEED_PASS?=Superadmin123!

seed:
	@echo "Seeding superadmin user..."
	@command -v psql >/dev/null 2>&1 || { echo "psql not found. Install PostgreSQL client."; exit 1; }
	@PGPASSWORD=$(DB_PASS) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) <<< "SET search_path TO public; INSERT INTO users (id, email, password_hash, roles, active, created_at, updated_at) VALUES ('00000000-0000-0000-0000-000000000001', '$(SEED_EMAIL)', '\$$2a\$$12\$$px2IB3Lw19v927vFTt4DjegR9P1ClwNqNY3bA9x/r0VE7RJM3dGbS', ARRAY['superadmin'], true, NOW(), NOW()) ON CONFLICT (email) DO UPDATE SET roles = ARRAY['superadmin'], active = true, password_hash = '\$$2a\$$12\$$px2IB3Lw19v927vFTt4DjegR9P1ClwNqNY3bA9x/r0VE7RJM3dGbS', updated_at = NOW();" && \
		echo "Superadmin seeded: $(SEED_EMAIL) / $(SEED_PASS)" || \
		echo "Failed to seed superadmin"

# Regenerate sqlc
sqlc:
	@echo "Regenerating sqlc..."
	@command -v sqlc >/dev/null 2>&1 || { echo "sqlc not found. Install with: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"; exit 1; }
	@sqlc generate
	@echo "sqlc regenerated"
