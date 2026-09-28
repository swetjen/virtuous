.PHONY: test fmt-check vet test-go test-go-norace test-example test-pgtype test-strict python-build python-publish python-clean publish

# Mirrors .github/workflows/ci.yaml: format check, vet, race tests, then the
# nested modules (example/* and pgtypetest/).
test: fmt-check vet test-go test-example test-pgtype

fmt-check:
	@offenders="$$(gofmt -l . | grep -v -e /testdata/ -e node_modules/ || true)"; \
	if [ -n "$$offenders" ]; then \
		echo "gofmt: the following files are not formatted:"; \
		echo "$$offenders"; \
		exit 1; \
	fi

vet:
	go vet ./...

# -race needs cgo, which means a C toolchain (gcc or clang) on PATH.
# Use test-go-norace on machines without one.
test-go:
	go test -race ./...

test-go-norace:
	go test ./...

test-example:
	@for d in example/*/; do \
		echo "== $$d"; \
		(cd "$$d" && go test ./...) || exit 1; \
	done

test-pgtype:
	cd pgtypetest && go test ./...

# Same as `test`, but the generated-client tests FAIL when node, tsc or uv are
# missing instead of skipping, exactly as they do in CI.
test-strict:
	CI=true $(MAKE) test

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
	@head="$$(git rev-parse HEAD)"; \
	runs="$$(gh run list --commit "$$head" --workflow ci.yaml --status success --json databaseId --jq 'length' 2>/dev/null)"; \
	if [ -z "$$runs" ] || [ "$$runs" = "0" ]; then \
		echo "CI has not passed for HEAD ($$head); push it and wait for the ci workflow to succeed"; exit 1; \
	fi
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
