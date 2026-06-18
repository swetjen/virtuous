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

## Notes

- `load_remote_module` requires a signed client and a trusted root key supplied by `root_public_key` or a `trust` callback/provider.
- `unsafe_load_module` preserves the old remote execution behavior under an explicit unsafe name for local/dev or fully trusted workflows.
- `load_module` was removed in Virtuous 0.0.56.
- `get_remote_hash` reads from `<url>.sha256`.
- Loaded modules expose `__virtuous_hash__` with the computed SHA-256 digest.

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

The callback may return `True` to accept the offered key, or return the trusted root public key value to compare against the signed client envelope.
