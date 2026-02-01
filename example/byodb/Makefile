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
