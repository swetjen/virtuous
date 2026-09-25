# Using Virtuous in Python

Verified loader for Virtuous generated Python clients.

## Install

```bash
pip install virtuous
```

## Usage

```python
from datetime import timedelta

from virtuous import load_remote_module, unsafe_load_module

module = load_remote_module(
    "https://api.example.com/client.gen.py",
    root_public_key="...",
    # Optional pins:
    expected_scope="https://api.example.com",  # authenticated origin scope must match
    max_age=timedelta(days=7),                 # signed issue time must be this fresh
    expected_hash="...",                       # exact SHA-256 of the client body
)
client = module.create_client("https://api.example.com")

# Explicit escape hatch for trusted local/dev workflows.
module = unsafe_load_module("http://localhost:8080/rpc/client.gen.py")
```

## How verification works

`load_remote_module` verifies the signed `client.gen.py` envelope before any
code executes:

1. The client body's SHA-256 is recomputed and must match `Virtuous-Body-SHA256`
   (and `expected_hash`, if pinned).
2. The root key signs a certificate binding the artifact key
   (`Virtuous-Artifact-Key-Cert`), which is verified first.
3. The artifact key signs a v2 manifest (`Virtuous-Manifest-Signature`) covering
   the origin scope, the issue time (`Virtuous-Issued-At`), and the body hash.
   The manifest is recomputed from the envelope headers and the independently
   computed body hash, so none of those values can be swapped on a signed
   artifact.
4. The root key itself is checked against `root_public_key`, or handed to your
   `trust` callback as `trust(scope, key_id, public_key)`.

Because the manifest covers the scope, the `scope` passed to the trust callback
and exposed as `module.__virtuous_scope__` is authenticated. An empty scope
stays empty — the fetch URL is never substituted for it.

### Pins

- `expected_hash` — hex SHA-256 the client body must match. Works for both v1
  and v2 envelopes. Raises `HashPinMismatchError`.
- `max_age` — seconds or a `timedelta`; the signed `Virtuous-Issued-At` must be
  no older. Requires a v2 manifest. Raises `ArtifactExpiredError`.
- `expected_scope` — the authenticated origin scope must match exactly.
  Requires a v2 manifest. Raises `ScopeMismatchError`.

All three errors subclass `RemoteClientVerificationError`.

### Legacy v1 envelopes

Clients signed by older servers (no `Virtuous-Manifest-Version` header) still
verify — the artifact key signature covers only the body — but emit a
`DeprecationWarning`: their origin scope and issue time are unauthenticated.
`max_age` and `expected_scope` cannot be satisfied by a v1 artifact and raise
`RemoteClientVerificationError`; `expected_hash` works on both. For v1 loads,
`module.__virtuous_scope__` is `None`.

## Notes

- Pass `root_public_key` directly or pass a `trust` callback/provider to handle
  root key approval dynamically.
- `unsafe_load_module` preserves the old remote execution behavior under an
  explicit unsafe name.
- `load_module` was removed in Virtuous 0.0.56.
- `get_remote_hash` reads from `<url>.sha256`.
- Loaded modules expose `__virtuous_hash__` (SHA-256 of the body),
  `__virtuous_scope__` (authenticated origin scope, or `None`), and
  `__virtuous_issued_at__` (signed issue time as a `datetime`, or `None`).

## Development

Run the tests with `uv`:

```bash
cd python_loader
uv run --with cryptography --with pytest pytest
```
