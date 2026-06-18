"""Basic import smoke test for virtuous."""

from virtuous import (
    RemoteClientVerificationError,
    get_remote_hash,
    hash_url,
    load_remote_module,
    load_remote_module_to_disk,
    unsafe_load_module,
    unsafe_load_module_to_disk,
)


def main() -> None:
    _ = RemoteClientVerificationError
    _ = get_remote_hash
    _ = hash_url
    _ = load_remote_module
    _ = load_remote_module_to_disk
    _ = unsafe_load_module
    _ = unsafe_load_module_to_disk
    print("ok")


if __name__ == "__main__":
    main()
