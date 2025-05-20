import hashlib
import re
import sys

from securetar import SecureTarFile, atomic_contents_add
from pathlib import Path, PurePath

RE_DIGITS = re.compile(r"\d+")
BUF_SIZE = 2**20 * 4  # 4MB


def password_to_key(password: str) -> bytes:
    """Generate a AES Key from password."""
    key: bytes = password.encode()
    for _ in range(100):
        key = hashlib.sha256(key).digest()
    return key[:16]


def key_to_iv(key: bytes) -> bytes:
    """Generate an iv from Key."""
    for _ in range(100):
        key = hashlib.sha256(key).digest()
    return key[:16]


def create_slug(name: str, date_str: str) -> str:
    """Generate a hash from repository."""
    key = f"{date_str} - {name}".lower().encode()
    return hashlib.sha1(key).hexdigest()[:8]


def main():
    compressed = True
    key = password_to_key("XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX")
    dir_for_encrypt = "./data/test_unencrypt"
    tmp_name = "./data"
    tar_name = f"test.tar{'.gz' if compressed else ''}"

    path = Path(tmp_name, tar_name)

    print(path)

    path_to_add = Path(dir_for_encrypt)

    with SecureTarFile(
        path,
        "w",
        key=key,
        gzip=True,
        bufsize=BUF_SIZE,
    ) as tar_file:
        atomic_contents_add(
            tar_file,
            path_to_add,
            excludes=[],
            arcname=".",
        )


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\n\n⚠️  Operation cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n❌ Unexpected error: {str(e)}")
        sys.exit(1)
