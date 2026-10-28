"""Run offline commands, record output, and check their teaching outcomes.
Run from repository root: python3 tools/check-examples.py
"""
import json
from pathlib import Path
import subprocess

root = Path(__file__).resolve().parents[1]
for number, expected in [(2, 1), (3, 0), (4, 0), (5, 0), (6, 0), (7, 0), (8, 1), (9, 0), (10, 0)]:
    folder = root / f"example{number}"
    result = subprocess.run(["go", "run", "."], cwd=folder, text=True, capture_output=True, timeout=60)
    assert result.returncode == expected, (number, result.stdout, result.stderr)
    if number in (7, 8):
        report = json.loads(result.stdout)
        if number == 7:
            assert report["valid"] and len(report["signatures"]) == 1
            name = "report.json"
        else:
            assert (report["accepted"], report["rejected"]) == (1, 2)
            assert [r["valid"] for r in report["documents"]] == [True, False, False]
            name = "batch-report.json"
        (folder / name).write_text(result.stdout)
    else:
        (folder / "expected-output.txt").write_text(result.stdout + result.stderr)
    print(f"example{number}: expected exit {expected}, observed {result.returncode}")

# Missing input remains an individual batch failure, preserving other results.
result = subprocess.run(["go", "run", ".", "signed.pdf", "missing.pdf"], cwd=root / "example8", capture_output=True, text=True, timeout=60)
assert result.returncode == 1
report = json.loads(result.stdout)
assert report["accepted"] == 1 and report["rejected"] == 1
assert report["documents"][1]["errors"]
print("batch missing-file case: passed")

# PDFs carry their own example number and case label, embedded before signing.
for number in range(1, 11):
    for pdf in (root / f"example{number}").glob("*.pdf"):
        data = pdf.read_bytes()
        assert f"Example {number} - ".encode() in data, str(pdf)
        assert b"Case:" in data and b"Expected result:" in data, str(pdf)
print("All example PDFs contain example, case and expected-result labels")
