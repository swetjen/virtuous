# Virtuous Loader

Stdlib-only loader for Virtuous runtime-generated Python clients.

## Usage

```python
from virtuous_loader import load_module, load_module_to_disk, get_remote_hash

module = load_module("https://api.example.com/client.gen.py")
client = module.create_client("https://api.example.com")

# Optional: write to disk and import later
module = load_module_to_disk(
    "https://api.example.com/client.gen.py",
    "/tmp/client.gen.py",
)

# Optional: fetch the server-provided hash
remote_hash = get_remote_hash("https://api.example.com/client.gen.py")
```

## Notes

- This loader executes remote Python code. Use trusted endpoints only.
- `get_remote_hash` reads from `<url>.sha256`.
- Loaded modules expose `__virtuous_hash__` with the computed SHA-256 digest.
