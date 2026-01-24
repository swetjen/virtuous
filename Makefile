.PHONY: test test-go test-js test-python test-ts

test: test-go test-js test-python test-ts

test-go:
	cd virtuous && go test ./...

test-js:
	@command -v node >/dev/null 2>&1 && echo "node present" || { echo "node not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1

test-python:
	@command -v python3 >/dev/null 2>&1 && echo "python3 present" || { echo "python3 not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1

test-ts:
	@command -v tsc >/dev/null 2>&1 && echo "tsc present" || { echo "tsc not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1
