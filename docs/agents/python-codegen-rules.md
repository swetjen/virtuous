---
title: Python Codegen Rules
description: "Namespace and collision rules the Python client generator must obey, with a regression checklist."
section: Agents
audience: agent
status: stable
related:
  - agents/client-codegen.md
---

# Python Codegen Rules

## Overview

Generated Python has one module namespace. DTO classes, transport classes, service classes, helper functions, imports, method names, fields, query params, path params, auth params, and locals must be generated as if they can collide.

## Rules

- Reserve generated runtime symbols before choosing DTO names. This includes `_VirtuousClient`, service class names, helper functions, imported modules, and the `create_client` factory.
- Keep generated transport/runtime implementation names private. Prefer names such as `_VirtuousClient` and `_APIService`; keep `create_client(...)` as the public entry point.
- Never emit raw API names into Python identifier positions. Sanitize class names, method names, fields, path params, query params, auth params, and locals.
- Preserve wire names separately from Python identifiers. For example, JSON or query name `from` should use Python identifier `from_` while still encoding and decoding the wire key `"from"`.
- Prefer API route/domain context over Go package paths for DTO names, including nested DTOs reached through request and response fields. Package-qualified names should be a last resort only when no route context is available.
- Do not rely on quoted forward references to hide collisions. `get_type_hints(...)` resolves annotations at runtime, so module bindings must point at the intended DTO classes.
- Runtime-test generated Python, not only `py_compile`. Tests should import the generated module, call generated methods with mocked `urlopen`, and assert decoded dataclass types.
- Run generated Python verification through `uv run --python 3.12 python ...` so contract tests use a consistent interpreter and fail clearly when `uv` is unavailable.
- Keep optional generated call surfaces keyword-only. Query params, auth overrides, and request bodies should not become ambiguous positional arguments.
- Fail locally before dispatch when generated clients know declared auth is missing.

## Regression Names

Python codegen tests should include hostile API names:

- Keywords: `from`, `class`, `try`, `for`, `else`, `with`, `async`, `await`, `lambda`.
- Builtins and common names: `list`, `dict`, `str`, `type`, `object`, `id`.
- Runtime-like names: `Client`, `APIClient`, `VirtuousClient`, `_VirtuousClient`, `create_client`, `_decode_value`, `_encode_value`, `_append_query`.
- Duplicate-safe names: `from`, `from_`, `from-`, `class`, `class_`.
- Param collisions: query/path/auth names such as `self`, `body`, `token_auth`, and `from`.

## Verification Checklist

Before finishing Python generator changes:

- Generate a real `client.gen.py`.
- Run `uv run --python 3.12 python -m py_compile client.gen.py`.
- Import the generated module with `importlib.util.spec_from_file_location(...)`.
- Instantiate `create_client(...)`.
- Mock `urllib.request.urlopen` and call at least one generated method.
- Assert decoded responses are dataclass DTOs, including nested DTOs.
- Assert reserved field names round-trip through `_encode_value(...)` and `_decode_value(...)`.
- Assert DTO names do not shadow runtime helpers or imports.
- Assert nested same-name DTOs from different packages use route/domain context, not package-qualified names.
- Run `make test`.
