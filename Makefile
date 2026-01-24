.PHONY: test test-go test-js test-python test-ts test-example python-build python-publish python-clean publish

test: test-go test-example test-js test-python test-ts

test-go:
	cd virtuous && go test ./...

test-example:
	cd example && go test ./...

test-js:
	@command -v node >/dev/null 2>&1 && echo "node present" || { echo "node not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1

test-python:
	@command -v python3 >/dev/null 2>&1 && echo "python3 present" || { echo "python3 not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1

test-ts:
	@command -v tsc >/dev/null 2>&1 && echo "tsc present" || { echo "tsc not found; skipping"; exit 0; }
	cd virtuous && go test ./... -run TestGeneratedClientsAreValid -count=1

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
	@git diff --quiet || { echo "working tree is dirty; commit before publishing"; exit 1; }
	@version="$$(cat VERSION)"; \
	tag="v$${version}"; \
	if git rev-parse "$${tag}" >/dev/null 2>&1; then \
		echo "tag $${tag} already exists"; exit 1; \
	fi; \
	git tag "$${tag}"; \
	git push origin "$${tag}"
