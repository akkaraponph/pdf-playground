"""Audit every generated PDF using pdfsign and independent Poppler checks.
Run from repository root: python3 tools/audit-pdfs.py
Requires Go, qpdf, pdftotext, pdfsig. Writes docs/pdf-audit.json.
"""
import json
import hashlib
from pathlib import Path
import subprocess
import tempfile

root = Path(__file__).resolve().parents[1]

def run(args, **kwargs):
    return subprocess.run(args, cwd=root, capture_output=True, text=True, timeout=60, **kwargs)

# Exact fixture manifest: (library verdict with the specified root, integrity, timestamp).
cases = {
    "example1/contract.pdf": (False, "unsigned", False),
    "example1/contract-signed.pdf": (False, "valid", True),
    "example1/contract-mock-signed.pdf": (True, "valid", True),
    "example2/contract-signed.pdf": (False, "valid", True),
    "example3/contract-signed.pdf": (False, "valid", True),
    "example6/baseline.pdf": (False, "valid", False),
    "example6/timestamped.pdf": (False, "valid", True),
    "example6/timestamped-live.pdf": (False, "valid", True),
}
for number in [4,5,6,7,8,9,10]:
    for name, expected in {
        "signed.pdf": (True, "valid", False),
        "tampered.pdf": (False, "digest_mismatch", False),
        "unsigned.pdf": (False, "unsigned", False),
    }.items():
        cases[f"example{number}/{name}"] = expected

actual = {str(p.relative_to(root)) for folder in root.glob("example[0-9]*") if folder.name.removeprefix("example").isdigit() for p in folder.glob("*.pdf")}
assert actual == set(cases), f"Manifest differs: {actual.symmetric_difference(cases)}"
reports = []
with tempfile.TemporaryDirectory(prefix="pdf-audit-") as directory:
    binary = str(Path(directory) / "report")
    built = run(["go", "build", "-o", binary, "./example7"])
    assert built.returncode == 0, built.stderr
    for path, (expected_valid, expected_integrity, timestamp) in sorted(cases.items()):
        syntax = run(["qpdf", "--check", path])
        assert syntax.returncode == 0, (path, syntax.stdout, syntax.stderr)
        text = run(["pdftotext", path, "-"])
        assert text.returncode == 0, (path, text.stderr)
        number = Path(path).parent.name.removeprefix("example")
        for label in [f"Example {number} -", "v1.0.0", "Case:", "Expected result:"]:
            assert label in text.stdout, (path, label)
        if expected_integrity == "digest_mismatch":
            assert "MODIFIED" in text.stdout and "900 USD" in text.stdout

        # Certificate pools are explicit and local; no system trust changes.
        trust = str(Path(path).parent / "certs" / "root.pem")
        if not (root / trust).exists() or Path(path).name in {"baseline.pdf", "timestamped.pdf", "timestamped-live.pdf"}:
            trust = "certs/root.crt.pem"
        report = run([binary, "-input", path, "-roots", trust])
        assert report.returncode in (0, 1), (path, report.stderr)
        data = json.loads(report.stdout)
        assert data["valid"] == expected_valid, (path, data)
        assert report.returncode == (0 if expected_valid else 1), (path, report.stderr)
        assert len(data["signatures"]) == (0 if expected_integrity == "unsigned" else 1), (path, data)
        if data["signatures"]:
            sig = data["signatures"][0]
            native_names = {"9": "สมชาย ใจดี", "10": "张伟"}
            if number in native_names:
                assert sig["signer"] == native_names[number], (path, sig)
            assert sig["timestamp_present"] == timestamp, (path, data)
            if expected_integrity == "digest_mismatch":
                assert any("Message digest mismatch" in e for e in sig.get("errors", [])), (path, data)
            elif not expected_valid:
                assert not sig["trusted_chain"], (path, data)

        # -nocert isolates cryptographic integrity from local certificate trust.
        independent = run(["pdfsig", "-nocert", path])
        out = independent.stdout
        if "Signature Validation: Signature is Valid." in out:
            integrity = "valid"
        elif "Signature Validation: Digest Mismatch." in out:
            integrity = "digest_mismatch"
        elif "does not contain any signatures" in out:
            integrity = "unsigned"
        else:
            raise AssertionError((path, independent.returncode, out, independent.stderr))
        assert integrity == expected_integrity, (path, out)
        if integrity != "unsigned":
            if number in {"9", "10"}:
                assert native_names[number] in out, (path, out)
            assert "Total document signed" in out, (path, out)
        data.update({"sha256": hashlib.sha256((root / path).read_bytes()).hexdigest(), "trust_roots": trust, "independent_integrity": integrity, "case_matches_expectation": True})
        reports.append(data)
        print(f"{path}: integrity={integrity}, pdfsign-valid={data['valid']}")

summary = {
    "scope": "All generated example PDFs; not an Adobe Acrobat validation run",
    "pdfsign_version": "v1.0.0",
    "external_revocation_fetching": False,
    "checks": ["qpdf syntax", "pdftotext labels", "pdfsign with explicit roots", "Poppler pdfsig integrity"],
    "total": len(reports),
    "matching_expected_cases": sum(r["case_matches_expectation"] for r in reports),
    "documents": reports,
}
(root / "docs" / "pdf-audit.json").write_text(json.dumps(summary, indent=2) + "\n")
print(f"PASS: all {len(reports)} cases match their expected outcomes; docs/pdf-audit.json")
