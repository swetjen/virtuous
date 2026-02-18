ifneq (,$(wildcard .env))
include .env
endif

GOOSE ?= goose
GOOSE_DRIVER ?= postgres
PG_SSLMODE ?= disable

PG_HOST ?= localhost
PG_PORT ?= 5432
PG_DB ?= byodb
PG_USER ?= byodb
PG_PASS ?= byodb

PG_ADMIN_USER ?= postgres
PG_ADMIN_PASS ?=

APP_DSN = host=$(PG_HOST) port=$(PG_PORT) dbname=$(PG_DB) user=$(PG_USER) password=$(PG_PASS) sslmode=$(PG_SSLMODE)
ADMIN_DSN = host=$(PG_HOST) port=$(PG_PORT) dbname=postgres user=$(PG_ADMIN_USER) password=$(PG_ADMIN_PASS) sslmode=$(PG_SSLMODE)
ADMIN_TABLE = $(shell printf '%s' "goose_db_version_admin_$(PG_DB)" | tr -c 'A-Za-z0-9_' '_')

ADMIN_TEMPLATE := db/sql/admin_schemas/202602160001_reset_and_provision.sql.tmpl
ADMIN_RENDERED_DIR := .generated/admin
ADMIN_RENDERED_MIGRATION := $(ADMIN_RENDERED_DIR)/202602160001_reset_and_provision.sql

.PHONY: test deps gen gen-sdk gen-web gen-all run agent-run init-db up down clean-admin-render

$(ADMIN_RENDERED_MIGRATION): $(ADMIN_TEMPLATE)
	@mkdir -p $(ADMIN_RENDERED_DIR)
	@sed \
		-e "s|__PG_DB__|$(PG_DB)|g" \
		-e "s|__PG_USER__|$(PG_USER)|g" \
		-e "s|__PG_PASS__|$(PG_PASS)|g" \
		$(ADMIN_TEMPLATE) > $(ADMIN_RENDERED_MIGRATION)

init-db: $(ADMIN_RENDERED_MIGRATION)
	@$(GOOSE) -table $(ADMIN_TABLE) -dir $(ADMIN_RENDERED_DIR) $(GOOSE_DRIVER) "$(ADMIN_DSN)" down || true
	@$(GOOSE) -table $(ADMIN_TABLE) -dir $(ADMIN_RENDERED_DIR) $(GOOSE_DRIVER) "$(ADMIN_DSN)" up

up:
	@$(GOOSE) -dir db/sql/schemas $(GOOSE_DRIVER) "$(APP_DSN)" up

down:
	@$(GOOSE) -dir db/sql/schemas $(GOOSE_DRIVER) "$(APP_DSN)" down

clean-admin-render:
	@rm -rf .generated

test:
	@echo "Running tests"
	go clean -testcache
	go test ./... -v

deps:
	@echo "Building deps"
	go install github.com/swetjen/virtuous@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/cespare/reflex@latest

gen:
	@echo "Generating sqlc output"
	cd db/sql && sqlc generate

gen-sdk:
	@echo "Generating SDKs"
	go run ./cmd/clients

gen-web:
	@echo "Building frontend assets"
	cd frontend-web && bun install && bun run build

gen-all:
	@echo "Generating sqlc output, SDKs, and frontend assets"
	make gen
	make gen-sdk
	make gen-web

run:
	reflex -r '(^|/)(cmd|config|db|deps|handlers|middleware|frontend-web/src)/.*\.(go|html|js|ts|tsx)$$|(^|/)router\.go$$|(^|/)frontend_embed\.go$$|(^|/)client_gen\.go$$|(^|/)\.env$$' -s -- sh -c "go run ./cmd/api/main.go"

agent-run:
	@bash -c ': > ERRORS; reflex -r '\''(^|/)(cmd|config|db|deps|handlers|middleware|frontend-web/src)/.*\.(go|html|js|ts|tsx)$$|(^|/)router\.go$$|(^|/)frontend_embed\.go$$|(^|/)client_gen\.go$$|(^|/)\.env$$'\'' -s -- sh -c "go run ./cmd/api/main.go" 2>&1 | tee ERRORS'
