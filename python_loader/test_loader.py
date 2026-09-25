"""Signed remote loader behavior tests."""

import base64
import hashlib
import struct
import tempfile
import unittest
from datetime import datetime, timedelta, timezone
from pathlib import Path

import virtuous
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from virtuous import (
    ArtifactExpiredError,
    HashPinMismatchError,
    RemoteClientVerificationError,
    ScopeMismatchError,
    load_remote_module,
    unsafe_load_module,
)


ARTIFACT_CERT_DOMAIN = b"virtuous-artifact-key-cert-v1\n"
BODY_SIGNATURE_DOMAIN = b"virtuous-python-client-body-v1\n"
MANIFEST_DOMAIN = b"virtuous-python-client-manifest-v2\n"
SCOPE = "https://api.example.test"


def _b64(data: bytes) -> str:
    return base64.b64encode(data).decode("ascii")


def _cert_payload(root_key_id: str, artifact_key_id: str, artifact_public: bytes) -> bytes:
    return (
        ARTIFACT_CERT_DOMAIN
        + b"root-key-id:"
        + root_key_id.encode("ascii")
        + b"\nartifact-key-id:"
        + artifact_key_id.encode("ascii")
        + b"\nartifact-public-key:"
        + _b64(artifact_public).encode("ascii")
        + b"\n"
    )


def _manifest(scope: str, issued_at: str, body_hash: str) -> bytes:
    payload = bytearray(MANIFEST_DOMAIN)
    for field in (scope, issued_at, body_hash):
        raw = field.encode("utf-8")
        payload += struct.pack(">Q", len(raw))
        payload += raw
    return bytes(payload)


def _now_rfc3339() -> str:
    now = datetime.now(timezone.utc).replace(microsecond=0)
    return now.isoformat().replace("+00:00", "Z")


def _envelope(fields: list[tuple[str, str]], body: bytes) -> bytes:
    envelope = "".join("# " + key + ": " + value + "\n" for key, value in fields)
    return (envelope + "# Virtuous-Signature-End\n").encode("utf-8") + body


def _signed_source_v2(
    body: bytes,
    scope: str = SCOPE,
    issued_at: str | None = None,
    cert: bytes | None = None,
    root_key: Ed25519PrivateKey | None = None,
    artifact_key: Ed25519PrivateKey | None = None,
    manifest_scope: str | None = None,
    manifest_issued_at: str | None = None,
) -> tuple[bytes, bytes]:
    """Build a v2 signed client, mirroring the Go envelope writer.

    manifest_scope/manifest_issued_at let tests sign a manifest that differs
    from the envelope headers.
    """
    if root_key is None:
        root_key = Ed25519PrivateKey.generate()
    if artifact_key is None:
        artifact_key = Ed25519PrivateKey.generate()
    if issued_at is None:
        issued_at = _now_rfc3339()
    root_public = root_key.public_key().public_bytes_raw()
    artifact_public = artifact_key.public_key().public_bytes_raw()
    if cert is None:
        cert = root_key.sign(_cert_payload("root", "artifact", artifact_public))
    body_hash = hashlib.sha256(body).hexdigest()
    manifest = _manifest(
        manifest_scope if manifest_scope is not None else scope,
        manifest_issued_at if manifest_issued_at is not None else issued_at,
        body_hash,
    )
    manifest_signature = artifact_key.sign(manifest)
    fields = [
        ("Virtuous-Signature-Version", "1"),
        ("Virtuous-Signature-Algorithm", "ed25519"),
        ("Virtuous-Manifest-Version", "2"),
        ("Virtuous-Origin-Scope", scope),
        ("Virtuous-Issued-At", issued_at),
        ("Virtuous-Root-Key-ID", "root"),
        ("Virtuous-Root-Public-Key", _b64(root_public)),
        ("Virtuous-Artifact-Key-ID", "artifact"),
        ("Virtuous-Artifact-Public-Key", _b64(artifact_public)),
        ("Virtuous-Artifact-Key-Cert", _b64(cert)),
        ("Virtuous-Body-SHA256", body_hash),
        ("Virtuous-Manifest-Signature", _b64(manifest_signature)),
    ]
    return _envelope(fields, body), root_public


def _signed_source_v1(
    body: bytes,
    cert: bytes | None = None,
    root_key: Ed25519PrivateKey | None = None,
    artifact_key: Ed25519PrivateKey | None = None,
    scope: str = "file://test/client.gen.py",
) -> tuple[bytes, bytes]:
    """Build a legacy v1 signed client (no manifest)."""
    if root_key is None:
        root_key = Ed25519PrivateKey.generate()
    if artifact_key is None:
        artifact_key = Ed25519PrivateKey.generate()
    root_public = root_key.public_key().public_bytes_raw()
    artifact_public = artifact_key.public_key().public_bytes_raw()
    if cert is None:
        cert = root_key.sign(_cert_payload("root", "artifact", artifact_public))
    body_signature = artifact_key.sign(BODY_SIGNATURE_DOMAIN + body)
    fields = [
        ("Virtuous-Signature-Version", "1"),
        ("Virtuous-Signature-Algorithm", "ed25519"),
        ("Virtuous-Origin-Scope", scope),
        ("Virtuous-Root-Key-ID", "root"),
        ("Virtuous-Root-Public-Key", _b64(root_public)),
        ("Virtuous-Artifact-Key-ID", "artifact"),
        ("Virtuous-Artifact-Public-Key", _b64(artifact_public)),
        ("Virtuous-Artifact-Key-Cert", _b64(cert)),
        ("Virtuous-Body-SHA256", hashlib.sha256(body).hexdigest()),
        ("Virtuous-Body-Signature", _b64(body_signature)),
    ]
    return _envelope(fields, body), root_public


def _replace_field(source: bytes, key: str, value: str) -> bytes:
    lines = source.splitlines(keepends=True)
    prefix = ("# " + key + ": ").encode("utf-8")
    for index, line in enumerate(lines):
        if line.startswith(prefix):
            lines[index] = prefix + value.encode("utf-8") + b"\n"
            return b"".join(lines)
    raise AssertionError("field not found: " + key)


def _duplicate_field(source: bytes, key: str) -> bytes:
    lines = source.splitlines(keepends=True)
    prefix = ("# " + key + ": ").encode("utf-8")
    for index, line in enumerate(lines):
        if line.startswith(prefix):
            lines.insert(index + 1, line)
            return b"".join(lines)
    raise AssertionError("field not found: " + key)


def _write_source(source: bytes) -> str:
    directory = tempfile.TemporaryDirectory()
    path = Path(directory.name) / "client.gen.py"
    path.write_bytes(source)
    _write_source.cleanups.append(directory)
    return path.as_uri()


_write_source.cleanups = []


class LoaderTest(unittest.TestCase):
    def test_load_module_is_removed(self) -> None:
        self.assertFalse(hasattr(virtuous, "load_module"))

    def test_unsafe_load_module_keeps_old_behavior(self) -> None:
        url = _write_source(b"VALUE = 42\n")

        module = unsafe_load_module(url)

        self.assertEqual(module.VALUE, 42)

    def test_load_remote_module_rejects_unsigned_before_exec(self) -> None:
        url = _write_source(b"raise RuntimeError('should not execute')\n")

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=b"0" * 32)

    def test_load_remote_module_accepts_valid_signed_client_with_root_key(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(url, root_public_key=_b64(root_public))

        self.assertEqual(module.VALUE, 42)
        self.assertEqual(module.__virtuous_scope__, SCOPE)
        self.assertIsNotNone(module.__virtuous_issued_at__)

    def test_load_remote_module_accepts_valid_signed_client_with_trust_callback(self) -> None:
        source, _ = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)
        seen: list[str] = []

        def trust(scope: str, key_id: str, public_key: str) -> str:
            seen.append(scope)
            return public_key

        module = load_remote_module(url, trust=trust)

        self.assertEqual(module.VALUE, 42)
        self.assertEqual(seen, [SCOPE])

    def test_load_remote_module_requires_trust_anchor(self) -> None:
        source, _ = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url)

    def test_load_remote_module_rejects_body_tamper_before_exec(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source + b"raise RuntimeError('should not execute')\n")

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_manifest_signature_tamper_before_exec(self) -> None:
        source, root_public = _signed_source_v2(b"raise RuntimeError('should not execute')\n")
        source = _replace_field(source, "Virtuous-Manifest-Signature", _b64(b"0" * 64))
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_rescoped_envelope(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        source = _replace_field(source, "Virtuous-Origin-Scope", "https://attacker.example.test")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_reissued_envelope(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        source = _replace_field(source, "Virtuous-Issued-At", "2030-01-01T00:00:00Z")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_manifest_signed_for_other_scope(self) -> None:
        source, root_public = _signed_source_v2(
            b"VALUE = 42\n",
            manifest_scope="https://attacker.example.test",
        )
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_manifest_signed_for_other_issued_at(self) -> None:
        source, root_public = _signed_source_v2(
            b"VALUE = 42\n",
            manifest_issued_at="2020-01-01T00:00:00Z",
        )
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_unknown_manifest_version(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        source = _replace_field(source, "Virtuous-Manifest-Version", "3")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_invalid_issued_at(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n", issued_at="not-a-time")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_wrong_root_key(self) -> None:
        source, _ = _signed_source_v2(b"VALUE = 42\n")
        wrong_root = Ed25519PrivateKey.generate().public_key().public_bytes_raw()
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=wrong_root)

    def test_load_remote_module_rejects_bad_artifact_cert_before_exec(self) -> None:
        source, root_public = _signed_source_v2(
            b"raise RuntimeError('should not execute')\n",
            cert=b"0" * 64,
        )
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_invalid_base64(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        source = _replace_field(source, "Virtuous-Artifact-Public-Key", "not base64")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_duplicate_critical_field(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        source = _duplicate_field(source, "Virtuous-Body-SHA256")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_accepts_artifact_key_rotation_under_same_root(self) -> None:
        root_key = Ed25519PrivateKey.generate()
        first_source, root_public = _signed_source_v2(b"VALUE = 42\n", root_key=root_key)
        second_source, _ = _signed_source_v2(b"VALUE = 43\n", root_key=root_key)
        first_url = _write_source(first_source)
        second_url = _write_source(second_source)

        first_module = load_remote_module(first_url, root_public_key=root_public)
        second_module = load_remote_module(second_url, root_public_key=root_public)

        self.assertEqual(first_module.VALUE, 42)
        self.assertEqual(second_module.VALUE, 43)

    def test_empty_scope_is_not_replaced_by_url(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n", scope="")
        url = _write_source(source)
        seen: list[str] = []

        def trust(scope: str, key_id: str, public_key: str) -> str:
            seen.append(scope)
            return public_key

        module = load_remote_module(url, trust=trust)

        self.assertEqual(module.VALUE, 42)
        self.assertEqual(module.__virtuous_scope__, "")
        self.assertEqual(seen, [""])


class PinTest(unittest.TestCase):
    def test_expected_hash_accepts_matching_body(self) -> None:
        body = b"VALUE = 42\n"
        source, root_public = _signed_source_v2(body)
        url = _write_source(source)

        module = load_remote_module(
            url,
            root_public_key=root_public,
            expected_hash=hashlib.sha256(body).hexdigest(),
        )

        self.assertEqual(module.VALUE, 42)

    def test_expected_hash_rejects_other_body(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertRaises(HashPinMismatchError):
            load_remote_module(
                url,
                root_public_key=root_public,
                expected_hash=hashlib.sha256(b"VALUE = 43\n").hexdigest(),
            )

    def test_max_age_accepts_fresh_artifact(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(url, root_public_key=root_public, max_age=300)

        self.assertEqual(module.VALUE, 42)

    def test_max_age_accepts_timedelta(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(
            url, root_public_key=root_public, max_age=timedelta(minutes=5)
        )

        self.assertEqual(module.VALUE, 42)

    def test_max_age_rejects_stale_artifact(self) -> None:
        stale = datetime.now(timezone.utc) - timedelta(days=2)
        issued_at = stale.replace(microsecond=0).isoformat().replace("+00:00", "Z")
        source, root_public = _signed_source_v2(b"VALUE = 42\n", issued_at=issued_at)
        url = _write_source(source)

        with self.assertRaises(ArtifactExpiredError):
            load_remote_module(url, root_public_key=root_public, max_age=3600)

    def test_expected_scope_accepts_matching_scope(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(
            url, root_public_key=root_public, expected_scope=SCOPE
        )

        self.assertEqual(module.VALUE, 42)

    def test_expected_scope_rejects_other_scope(self) -> None:
        source, root_public = _signed_source_v2(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertRaises(ScopeMismatchError):
            load_remote_module(
                url,
                root_public_key=root_public,
                expected_scope="https://other.example.test",
            )


class V1BackCompatTest(unittest.TestCase):
    def test_v1_envelope_loads_with_deprecation_warning(self) -> None:
        source, root_public = _signed_source_v1(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertWarns(DeprecationWarning):
            module = load_remote_module(url, root_public_key=root_public)

        self.assertEqual(module.VALUE, 42)
        self.assertIsNone(module.__virtuous_scope__)
        self.assertIsNone(module.__virtuous_issued_at__)

    def test_v1_envelope_rejects_body_tamper(self) -> None:
        source, root_public = _signed_source_v1(b"VALUE = 42\n")
        url = _write_source(source + b"raise RuntimeError('should not execute')\n")

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_v1_envelope_rejects_body_signature_tamper(self) -> None:
        source, root_public = _signed_source_v1(b"raise RuntimeError('should not execute')\n")
        source = _replace_field(source, "Virtuous-Body-Signature", _b64(b"0" * 64))
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_v1_envelope_cannot_satisfy_max_age(self) -> None:
        source, root_public = _signed_source_v1(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public, max_age=10**9)

    def test_v1_envelope_cannot_satisfy_expected_scope(self) -> None:
        source, root_public = _signed_source_v1(b"VALUE = 42\n", scope=SCOPE)
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public, expected_scope=SCOPE)

    def test_v1_envelope_expected_hash_still_works(self) -> None:
        body = b"VALUE = 42\n"
        source, root_public = _signed_source_v1(body)
        url = _write_source(source)

        with self.assertWarns(DeprecationWarning):
            module = load_remote_module(
                url,
                root_public_key=root_public,
                expected_hash=hashlib.sha256(body).hexdigest(),
            )
        self.assertEqual(module.VALUE, 42)

        with self.assertRaises(HashPinMismatchError):
            load_remote_module(
                url,
                root_public_key=root_public,
                expected_hash=hashlib.sha256(b"VALUE = 43\n").hexdigest(),
            )

    def test_v1_trust_callback_never_receives_url(self) -> None:
        source, _ = _signed_source_v1(b"VALUE = 42\n", scope="")
        url = _write_source(source)
        seen: list[str] = []

        def trust(scope: str, key_id: str, public_key: str) -> str:
            seen.append(scope)
            return public_key

        module = load_remote_module(url, trust=trust)

        self.assertEqual(module.VALUE, 42)
        self.assertEqual(seen, [""])


if __name__ == "__main__":
    unittest.main()
