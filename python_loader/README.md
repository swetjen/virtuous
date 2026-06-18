# Using Virtuous in Python

Verified loader for Virtuous generated Python clients.

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

# Explicit escape hatch for trusted local/dev workflows.
module = unsafe_load_module("http://localhost:8080/rpc/client.gen.py")
```

## Notes

- `load_remote_module` verifies the signed `client.gen.py` envelope before executing code.
- Pass `root_public_key` directly or pass a `trust` callback/provider to handle root key approval dynamically.
- `unsafe_load_module` preserves the old remote execution behavior under an explicit unsafe name.
- `load_module` was removed in Virtuous 0.0.56.
- `get_remote_hash` reads from `<url>.sha256`.
- Loaded modules expose `__virtuous_hash__` with the computed SHA-256 digest.
