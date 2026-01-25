test:
	@echo "Running tests"
	go clean -testcache
	go test ./... -v

deps:
	@echo "Building deps"
	go install github.com/swetjen/virtuous@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/cespare/reflex@latest

run:
	reflex -r ".*\.(go,html,js,ts)" -s -- sh -c "go run ./cmd/api/main.go"
