"""Loader for Virtuous Python clients."""

from importlib import machinery, util
import base64
import hashlib
import os
import sys
import types
from typing import Any, Callable, Optional
from urllib import request

from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey


DEFAULT_TIMEOUT = 30
SIGNATURE_VERSION = "1"
SIGNATURE_ALGORITHM = "ed25519"
ARTIFACT_CERT_DOMAIN = b"virtuous-artifact-key-cert-v1\n"
BODY_SIGNATURE_DOMAIN = b"virtuous-python-client-body-v1\n"


class RemoteClientVerificationError(ValueError):
    """Raised when a remote Virtuous client cannot be verified."""


TrustCallback = Callable[[str, str, str], Any]


def _fetch(url: str, timeout: float = DEFAULT_TIMEOUT) -> bytes:
    with request.urlopen(url, timeout=timeout) as resp:
        return resp.read()


def _hash_bytes(data: bytes) -> str:
    digest = hashlib.sha256(data).hexdigest()
    return digest


def hash_url(url: str) -> str:
    return url + ".sha256"


def get_remote_hash(url: str) -> str:
    data = _fetch(hash_url(url))
    text = data.decode("utf-8").strip()
    if not text:
        raise ValueError("empty hash response")
    return text.split()[0]


def _module_name(module_name: Optional[str], digest: str) -> str:
    if module_name:
        return module_name
    return "virtuous_client_" + digest[:8]


def _exec_module(source: bytes, origin: str, module_name: Optional[str]) -> types.ModuleType:
    digest = _hash_bytes(source)
    name = _module_name(module_name, digest)
    module = types.ModuleType(name)
    module.__file__ = origin
    module.__spec__ = machinery.ModuleSpec(name=name, loader=None, origin=origin)
    module.__virtuous_hash__ = digest
    sys.modules[name] = module
    code = compile(source, origin, "exec")
    exec(code, module.__dict__)
    return module


def unsafe_load_module(
    url: str,
    module_name: Optional[str] = None,
    timeout: float = DEFAULT_TIMEOUT,
) -> types.ModuleType:
    """Fetch and execute a Virtuous Python client without signature checks."""
    source = _fetch(url, timeout=timeout)
    return _exec_module(source, url, module_name)


def unsafe_load_module_to_disk(
    url: str,
    path: str,
    module_name: Optional[str] = None,
    timeout: float = DEFAULT_TIMEOUT,
) -> types.ModuleType:
    """Fetch, write, and execute a Virtuous Python client without signature checks."""
    source = _fetch(url, timeout=timeout)
    _write_source(path, source)
    return _load_module_from_disk(path, module_name, _hash_bytes(source))


def load_remote_module(
    url: str,
    module_name: Optional[str] = None,
    root_public_key: Optional[str | bytes] = None,
    trust: Optional[TrustCallback | object] = None,
    timeout: float = DEFAULT_TIMEOUT,
) -> types.ModuleType:
    """Fetch, verify, and execute a signed Virtuous Python client."""
    source = _fetch(url, timeout=timeout)
    verified = _verify_remote_source(source, url, root_public_key, trust)
    return _exec_module(verified.body, url, module_name)


def load_remote_module_to_disk(
    url: str,
    path: str,
    module_name: Optional[str] = None,
    root_public_key: Optional[str | bytes] = None,
    trust: Optional[TrustCallback | object] = None,
    timeout: float = DEFAULT_TIMEOUT,
) -> types.ModuleType:
    """Fetch, verify, write, and execute a signed Virtuous Python client."""
    source = _fetch(url, timeout=timeout)
    verified = _verify_remote_source(source, url, root_public_key, trust)
    _write_source(path, source)
    return _load_module_from_disk(path, module_name, _hash_bytes(verified.body))


class _VerifiedSource:
    def __init__(self, body: bytes, fields: dict[str, str]) -> None:
        self.body = body
        self.fields = fields


def _verify_remote_source(
    source: bytes,
    url: str,
    root_public_key: Optional[str | bytes],
    trust: Optional[TrustCallback | object],
) -> _VerifiedSource:
    fields, body = _split_signature_envelope(source)
    _require_field(fields, "Virtuous-Signature-Version", SIGNATURE_VERSION)
    _require_field(fields, "Virtuous-Signature-Algorithm", SIGNATURE_ALGORITHM)

    body_hash = _require_present(fields, "Virtuous-Body-SHA256")
    if _hash_bytes(body) != body_hash:
        raise RemoteClientVerificationError("Virtuous client body hash mismatch")

    root_key_id = _require_present(fields, "Virtuous-Root-Key-ID")
    artifact_key_id = _require_present(fields, "Virtuous-Artifact-Key-ID")
    root_public = _decode_b64_field(fields, "Virtuous-Root-Public-Key")
    artifact_public = _decode_b64_field(fields, "Virtuous-Artifact-Public-Key")
    artifact_cert = _decode_b64_field(fields, "Virtuous-Artifact-Key-Cert")
    body_signature = _decode_b64_field(fields, "Virtuous-Body-Signature")
    scope = fields.get("Virtuous-Origin-Scope") or url

    _check_root_trust(scope, root_key_id, root_public, root_public_key, trust)
    _verify_signature(
        root_public,
        _artifact_cert_payload(root_key_id, artifact_key_id, artifact_public),
        artifact_cert,
        "artifact key certificate",
    )
    _verify_signature(
        artifact_public,
        BODY_SIGNATURE_DOMAIN + body,
        body_signature,
        "client body signature",
    )
    return _VerifiedSource(body, fields)


def _split_signature_envelope(source: bytes) -> tuple[dict[str, str], bytes]:
    fields: dict[str, str] = {}
    offset = 0
    found_end = False
    for line in source.splitlines(keepends=True):
        offset += len(line)
        if not line.startswith(b"# "):
            break
        text = line[2:].decode("utf-8", errors="strict").strip()
        if text == "Virtuous-Signature-End":
            found_end = True
            break
        if not text.startswith("Virtuous-"):
            continue
        if ":" not in text:
            continue
        key, value = text.split(":", 1)
        key = key.strip()
        if key in fields:
            raise RemoteClientVerificationError("duplicate Virtuous signature field: " + key)
        fields[key] = value.strip()
    if not found_end:
        raise RemoteClientVerificationError("missing Virtuous signature envelope")
    return fields, source[offset:]


def _require_present(fields: dict[str, str], key: str) -> str:
    value = fields.get(key)
    if not value:
        raise RemoteClientVerificationError("missing Virtuous signature field: " + key)
    return value


def _require_field(fields: dict[str, str], key: str, expected: str) -> None:
    actual = _require_present(fields, key)
    if actual != expected:
        raise RemoteClientVerificationError("unsupported " + key + ": " + actual)


def _decode_b64_field(fields: dict[str, str], key: str) -> bytes:
    value = _require_present(fields, key)
    try:
        return base64.b64decode(value, validate=True)
    except ValueError as exc:
        raise RemoteClientVerificationError("invalid base64 in " + key) from exc


def _check_root_trust(
    scope: str,
    key_id: str,
    offered_public_key: bytes,
    root_public_key: Optional[str | bytes],
    trust: Optional[TrustCallback | object],
) -> None:
    if root_public_key is not None:
        trusted = _decode_public_key_value(root_public_key)
        if trusted != offered_public_key:
            raise RemoteClientVerificationError("root public key mismatch")
        return
    if trust is None:
        raise RemoteClientVerificationError("root public key or trust callback is required")
    offered = base64.b64encode(offered_public_key).decode("ascii")
    result: Any
    if callable(trust):
        result = trust(scope, key_id, offered)
    elif hasattr(trust, "trust_root_key"):
        result = trust.trust_root_key(scope, key_id, offered)
    elif hasattr(trust, "root_public_key"):
        result = trust.root_public_key(scope, key_id, offered)
    else:
        raise RemoteClientVerificationError("invalid trust provider")
    if isinstance(result, bool):
        if not result:
            raise RemoteClientVerificationError("root public key rejected by trust provider")
        return
    if isinstance(result, (bytes, str)):
        trusted = _decode_public_key_value(result)
        if trusted != offered_public_key:
            raise RemoteClientVerificationError("root public key mismatch")
        return
    raise RemoteClientVerificationError("trust provider did not approve root public key")


def _decode_public_key_value(value: str | bytes) -> bytes:
    if isinstance(value, bytes):
        if len(value) == 32:
            return value
        try:
            return base64.b64decode(value, validate=True)
        except ValueError as exc:
            raise RemoteClientVerificationError("invalid root public key") from exc
    try:
        return base64.b64decode(value, validate=True)
    except ValueError as exc:
        raise RemoteClientVerificationError("invalid root public key") from exc


def _artifact_cert_payload(root_key_id: str, artifact_key_id: str, artifact_public: bytes) -> bytes:
    artifact_b64 = base64.b64encode(artifact_public).decode("ascii")
    text = (
        ARTIFACT_CERT_DOMAIN.decode("ascii")
        + "root-key-id:"
        + root_key_id
        + "\nartifact-key-id:"
        + artifact_key_id
        + "\nartifact-public-key:"
        + artifact_b64
        + "\n"
    )
    return text.encode("ascii")


def _verify_signature(public_key: bytes, payload: bytes, signature: bytes, label: str) -> None:
    try:
        Ed25519PublicKey.from_public_bytes(public_key).verify(signature, payload)
    except (ValueError, InvalidSignature) as exc:
        raise RemoteClientVerificationError("invalid " + label) from exc


def _write_source(path: str, source: bytes) -> None:
    dir_path = os.path.dirname(os.path.abspath(path))
    if dir_path:
        os.makedirs(dir_path, exist_ok=True)
    with open(path, "wb") as handle:
        handle.write(source)


def _load_module_from_disk(path: str, module_name: Optional[str], digest: str) -> types.ModuleType:
    name = _module_name(module_name, digest)
    spec = util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise ImportError("unable to create module spec for path")
    module = util.module_from_spec(spec)
    module.__virtuous_hash__ = digest
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module
