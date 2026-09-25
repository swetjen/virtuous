---
title: Python Loader
description: "The Python loader that verifies and loads a signed runtime-generated client.gen.py."
section: Python Loader
audience: both
status: stable
---

# Python loader

## Overview

Virtuous ships a Python loader for runtime-generated Python clients. The supported dynamic path fetches `client.gen.py`, verifies its embedded Ed25519 signing envelope, and only then loads it as a module.

## Install

```bash
pip install virtuous
```

## Usage

```python
from virtuous import load_remote_module, unsafe_load_module

module = load_remote_module(
    "https://api.example.com/client.gen.py",
    root_public_key="...",
)
client = module.create_client("https://api.example.com")

module = unsafe_load_module("http://localhost:8080/rpc/client.gen.py")
```

## Pinning and freshness

The v2 signing envelope covers the origin scope and an issued-at timestamp, so
these can be enforced at load time:

```python
module = load_remote_module(
    "https://api.example.com/client.gen.py",
    root_public_key="...",
    expected_scope="billing-api",   # exact match against the signed scope
    max_age=86400,                  # seconds (or a timedelta) since Issued-At
    expected_hash="<sha256 hex>",   # pin an exact artifact
)
```

Each pin raises a dedicated `RemoteClientVerificationError` subclass on
violation (`ScopeMismatchError`, `ArtifactExpiredError`, `HashPinMismatchError`).
v1 envelopes still verify with a `DeprecationWarning`, but cannot satisfy
`expected_scope` or `max_age` (their scope and age are unauthenticated).

## Notes

- `load_remote_module` requires a signed client and a trusted root key supplied by `root_public_key` or a `trust` callback/provider.
- `unsafe_load_module` preserves the old remote execution behavior under an explicit unsafe name for local/dev or fully trusted workflows.
- `load_module` was removed in Virtuous 0.0.56.
- `get_remote_hash` reads from `<url>.sha256` — a change detector only, not an integrity mechanism.
- Loaded modules expose `__virtuous_hash__` (computed SHA-256), `__virtuous_scope__` (the signed scope; `None` for v1), and `__virtuous_issued_at__` (`datetime`, or `None` for v1).

## Trust callbacks

Use a callback when the root key comes from an application-managed source such as an environment variable, database, or secrets manager:

```python
def trust_root(scope: str, key_id: str, offered_public_key: str):
    return lookup_root_key(scope, key_id)

module = load_remote_module(
    "https://api.example.com/client.gen.py",
    trust=trust_root,
)
```

The callback may return `True` to accept the offered key, or return the trusted root public key value to compare against the signed client envelope. For v2 envelopes the `scope` argument is authenticated by the artifact's signed manifest; for v1 envelopes it is unauthenticated and should not be used as a trust decision on its own.
