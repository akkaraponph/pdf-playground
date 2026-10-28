"""Ensure credentials are contained in certs folders, ignored, and absent from Git's index."""
from pathlib import Path
import re
import subprocess

root = Path(__file__).resolve().parents[1]
extensions = {".pem", ".key", ".p12", ".pfx", ".p8", ".der", ".crt", ".cer", ".jks", ".keystore"}
indexed = subprocess.check_output(["git", "ls-files", "--cached", "-z"], cwd=root).decode().split("\0")
for name in filter(None, indexed):
    assert Path(name).suffix.lower() not in extensions, f"Credential is in Git index: {name}"

count = 0
for path in root.rglob("*"):
    if ".git" in path.relative_to(root).parts or not path.is_file():
        continue
    relative = path.relative_to(root)
    if path.suffix.lower() in extensions:
        assert "certs" in relative.parts[:-1], f"Credential outside a certs/ folder: {relative}"
        cert_dir = root.joinpath(*relative.parts[:relative.parts.index("certs") + 1])
        assert (cert_dir / ".gitignore").is_file(), f"Missing local .gitignore: {cert_dir}"
        ignored = subprocess.run(["git", "check-ignore", "-q", str(relative)], cwd=root)
        assert ignored.returncode == 0, f"Credential is not ignored: {relative}"
        count += 1
    if "certs" not in relative.parts[:-1]:
        assert not re.search(rb"(?m)^-----BEGIN (?:RSA |EC |ENCRYPTED |OPENSSH )?PRIVATE KEY-----", path.read_bytes()), f"Private key outside certs/: {relative}"
print(f"Credential policy passed: {count} credential files in ignored certs/ folders; none indexed by Git")
