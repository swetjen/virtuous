# Basic test client for the generated Python SDK.

import importlib.util
from pathlib import Path


def load_client_module():
    module_path = Path(__file__).with_name("client.gen.py")
    spec = importlib.util.spec_from_file_location("client_gen", module_path)
    if spec is None or spec.loader is None:
        raise RuntimeError("could not load client.gen.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main() -> None:
    module = load_client_module()
    client = module.create_client("http://localhost:8000")
    try:
        response = client.States.getByCode("mn")
    except Exception as exc:
        print(f"request failed: {exc}")
        return
    print(response)


if __name__ == "__main__":
    main()
