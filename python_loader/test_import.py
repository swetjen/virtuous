"""Basic import smoke test for virtuous."""

from virtuous import get_remote_hash, hash_url, load_module, load_module_to_disk


def main() -> None:
    _ = get_remote_hash
    _ = hash_url
    _ = load_module
    _ = load_module_to_disk
    print("ok")


if __name__ == "__main__":
    main()
