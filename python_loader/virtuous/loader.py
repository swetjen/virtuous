"""Runtime loader for Virtuous Python clients."""

from importlib import machinery, util
import sys
import types
from typing import Optional
from urllib import request
import hashlib
import os


def _fetch(url: str) -> bytes:
    with request.urlopen(url) as resp:
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


def load_module(url: str, module_name: Optional[str] = None) -> types.ModuleType:
    """Load a Virtuous Python client module from a URL."""
    source = _fetch(url)
    digest = _hash_bytes(source)
    name = _module_name(module_name, digest)
    module = types.ModuleType(name)
    module.__file__ = url
    module.__spec__ = machinery.ModuleSpec(name=name, loader=None, origin=url)
    module.__virtuous_hash__ = digest
    sys.modules[name] = module
    code = compile(source, url, "exec")
    exec(code, module.__dict__)
    return module


def load_module_to_disk(
    url: str,
    path: str,
    module_name: Optional[str] = None,
) -> types.ModuleType:
    """Fetch a Virtuous Python client module and write it to disk before loading."""
    source = _fetch(url)
    digest = _hash_bytes(source)
    name = _module_name(module_name, digest)
    dir_path = os.path.dirname(os.path.abspath(path))
    if dir_path:
        os.makedirs(dir_path, exist_ok=True)
    with open(path, "wb") as handle:
        handle.write(source)
    spec = util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise ImportError("unable to create module spec for path")
    module = util.module_from_spec(spec)
    module.__virtuous_hash__ = digest
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module
