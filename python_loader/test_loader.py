"""Signed remote loader behavior tests."""

import base64
import hashlib
import tempfile
import unittest
from pathlib import Path

import virtuous
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from virtuous import RemoteClientVerificationError, load_remote_module, unsafe_load_module


ARTIFACT_CERT_DOMAIN = b"virtuous-artifact-key-cert-v1\n"
BODY_SIGNATURE_DOMAIN = b"virtuous-python-client-body-v1\n"


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


def _signed_source(
    body: bytes,
    cert: bytes | None = None,
    root_key: Ed25519PrivateKey | None = None,
    artifact_key: Ed25519PrivateKey | None = None,
) -> tuple[bytes, bytes]:
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
        ("Virtuous-Origin-Scope", "file://test/client.gen.py"),
        ("Virtuous-Root-Key-ID", "root"),
        ("Virtuous-Root-Public-Key", _b64(root_public)),
        ("Virtuous-Artifact-Key-ID", "artifact"),
        ("Virtuous-Artifact-Public-Key", _b64(artifact_public)),
        ("Virtuous-Artifact-Key-Cert", _b64(cert)),
        ("Virtuous-Body-SHA256", hashlib.sha256(body).hexdigest()),
        ("Virtuous-Body-Signature", _b64(body_signature)),
    ]
    envelope = "".join("# " + key + ": " + value + "\n" for key, value in fields)
    return (envelope + "# Virtuous-Signature-End\n").encode("utf-8") + body, root_public


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
        source, root_public = _signed_source(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(url, root_public_key=_b64(root_public))

        self.assertEqual(module.VALUE, 42)

    def test_load_remote_module_accepts_valid_signed_client_with_trust_callback(self) -> None:
        source, _ = _signed_source(b"VALUE = 42\n")
        url = _write_source(source)

        module = load_remote_module(url, trust=lambda scope, key_id, public_key: public_key)

        self.assertEqual(module.VALUE, 42)

    def test_load_remote_module_requires_trust_anchor(self) -> None:
        source, _ = _signed_source(b"VALUE = 42\n")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url)

    def test_load_remote_module_rejects_body_tamper_before_exec(self) -> None:
        source, root_public = _signed_source(b"VALUE = 42\n")
        url = _write_source(source + b"raise RuntimeError('should not execute')\n")

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_body_signature_tamper_before_exec(self) -> None:
        source, root_public = _signed_source(b"raise RuntimeError('should not execute')\n")
        source = _replace_field(source, "Virtuous-Body-Signature", _b64(b"0" * 64))
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_wrong_root_key(self) -> None:
        source, _ = _signed_source(b"VALUE = 42\n")
        wrong_root = Ed25519PrivateKey.generate().public_key().public_bytes_raw()
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=wrong_root)

    def test_load_remote_module_rejects_bad_artifact_cert_before_exec(self) -> None:
        source, root_public = _signed_source(
            b"raise RuntimeError('should not execute')\n",
            cert=b"0" * 64,
        )
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_invalid_base64(self) -> None:
        source, root_public = _signed_source(b"VALUE = 42\n")
        source = _replace_field(source, "Virtuous-Artifact-Public-Key", "not base64")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_rejects_duplicate_critical_field(self) -> None:
        source, root_public = _signed_source(b"VALUE = 42\n")
        source = _duplicate_field(source, "Virtuous-Body-SHA256")
        url = _write_source(source)

        with self.assertRaises(RemoteClientVerificationError):
            load_remote_module(url, root_public_key=root_public)

    def test_load_remote_module_accepts_artifact_key_rotation_under_same_root(self) -> None:
        root_key = Ed25519PrivateKey.generate()
        first_source, root_public = _signed_source(b"VALUE = 42\n", root_key=root_key)
        second_source, _ = _signed_source(b"VALUE = 43\n", root_key=root_key)
        first_url = _write_source(first_source)
        second_url = _write_source(second_source)

        first_module = load_remote_module(first_url, root_public_key=root_public)
        second_module = load_remote_module(second_url, root_public_key=root_public)

        self.assertEqual(first_module.VALUE, 42)
        self.assertEqual(second_module.VALUE, 43)


if __name__ == "__main__":
    unittest.main()
