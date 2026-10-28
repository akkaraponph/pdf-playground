#!/usr/bin/env bash
# Rebuild every example PDF, including live B-T signatures, then audit each case.
set -euo pipefail
cd "$(dirname "$0")/.."
tsa_url=${1:?Usage: bash tools/retest-all.sh https://your-tsa.example/tsr}
for program in go python3 qpdf pdfsig pdftotext; do
  command -v "$program" >/dev/null || { echo "Missing prerequisite: $program" >&2; exit 1; }
done
if [[ ! -f certs/signer.key.pem || ! -f certs/signer.crt.pem || ! -f certs/root.crt.pem ]]; then
  go run ./tools/generate-certs
fi
go run ./tools/generate-fixtures -tsa "$tsa_url"
go run ./example1 \
  -input example1/contract.pdf -output example1/contract-mock-signed.pdf \
  -certificate certs/signer.crt.pem -private-key certs/signer.key.pem \
  -tsa "$tsa_url" | tee example1/expected-output.txt
(
  cd example6
  go run . -tsa "$tsa_url" | tee live-output.txt
)
for example in example9 example10; do
  (
    cd "$example"
    go run . | tee expected-output.txt
  )
done
python3 tools/check-examples.py
go test -count=1 -cover ./...
python3 tools/check-credentials.py
go vet ./...
python3 tools/audit-pdfs.py
