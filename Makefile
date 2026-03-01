.PHONY: test test-go test-js test-python test-ts test-example python-build python-publish python-clean publish

test: test-go test-example test-js test-python test-ts

test-go:
	go test ./...

test-example:
	cd example/basic-combined && go test ./...
	cd example/basic-httpapi && go test ./...
	cd example/basic-rpc && go test ./...
	cd example/byodb && go test ./...

test-js:
	@command -v node >/dev/null 2>&1 && echo "node present" || { echo "node not found; skipping"; exit 0; }
	go test ./... -run TestGeneratedClientsAreValid -count=1

test-python:
	@command -v python3 >/dev/null 2>&1 && echo "python3 present" || { echo "python3 not found; skipping"; exit 0; }
	go test ./... -run TestGeneratedClientsAreValid -count=1

test-ts:
	@command -v tsc >/dev/null 2>&1 && echo "tsc present" || { echo "tsc not found; skipping"; exit 0; }
	go test ./... -run TestGeneratedClientsAreValid -count=1

ROOT_DIR := $(abspath .)
PYTHON_LOADER_DIR := $(ROOT_DIR)/python_loader
VENV_BIN := $(ROOT_DIR)/.venv/bin
PYTHON := $(VENV_BIN)/python
TWINE := $(VENV_BIN)/twine
UV := uv
PYTHON_DEPS := build twine setuptools wheel

.PHONY: python-venv
.PHONY: python-deps

python-venv:
	@if [ ! -x "$(PYTHON)" ]; then \
		echo "Creating .venv with uv..."; \
		$(UV) venv $(ROOT_DIR)/.venv; \
	fi

python-deps:
	$(MAKE) python-venv
	$(UV) pip install $(PYTHON_DEPS)

python-build:
	$(MAKE) python-deps
	cd $(PYTHON_LOADER_DIR) && $(PYTHON) -m build --no-isolation

python-clean:
	rm -rf $(PYTHON_LOADER_DIR)/dist $(PYTHON_LOADER_DIR)/build $(PYTHON_LOADER_DIR)/*.egg-info

python-publish:
	$(MAKE) python-deps
	$(MAKE) python-clean
	cd $(PYTHON_LOADER_DIR) && $(PYTHON) -m build --no-isolation
	cd $(PYTHON_LOADER_DIR) && $(TWINE) upload dist/*

publish:
	@[ -z "$$(git status --porcelain)" ] || { echo "working tree is dirty; commit before publishing"; exit 1; }
	@[ "$$(git branch --show-current)" = "main" ] || { echo "publish must run from main"; exit 1; }
	@command -v gh >/dev/null 2>&1 || { echo "gh CLI is required for publishing"; exit 1; }
	@version="$$(cat VERSION)"; \
	tag="v$${version}"; \
	notes_file="$$(mktemp)"; \
	trap 'rm -f "$$notes_file"' EXIT; \
	awk -v version="$$version" '\
		$$0 == "## " version { found=1; next } \
		found && /^## / { exit } \
		found { print } \
	' CHANGELOG.md > "$$notes_file"; \
	if [ ! -s "$$notes_file" ]; then \
		echo "missing CHANGELOG entry for $${version}"; exit 1; \
	fi; \
	if ! git rev-parse "$${tag}" >/dev/null 2>&1; then \
		git tag "$${tag}"; \
	fi; \
	if ! git ls-remote --tags origin "$${tag}" | grep -q "$${tag}"; then \
		git push origin "$${tag}"; \
	fi; \
	if gh release view "$${tag}" >/dev/null 2>&1; then \
		echo "GitHub release $${tag} already exists"; exit 1; \
	fi; \
	gh release create "$${tag}" --title "$${tag}" --notes-file "$$notes_file"
