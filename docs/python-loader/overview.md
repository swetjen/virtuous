# Python loader

## Overview

Virtuous ships a zero-dependency Python loader for runtime-generated Python clients. It fetches `client.gen.py` from a Virtuous server and loads it as a module.

## Install

```bash
pip install virtuous
```

## Usage

```python
from virtuous import load_module, load_module_to_disk, get_remote_hash

module = load_module("https://api.example.com/client.gen.py")
client = module.create_client("https://api.example.com")

module = load_module_to_disk(
    "https://api.example.com/client.gen.py",
    "/tmp/client.gen.py",
)

remote_hash = get_remote_hash("https://api.example.com/client.gen.py")
```

## Notes

- The loader executes remote Python code. Use trusted endpoints only.
- `get_remote_hash` reads from `<url>.sha256`.
- Loaded modules expose `__virtuous_hash__` with the computed SHA-256 digest.
