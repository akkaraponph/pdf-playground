# Example 7: Export a JSON verification report

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example7
go run . > report.json
```

Writes a JSON report with `valid: true`, the signer, chain trust, timestamp presence, errors, and warnings. Exit 1 means the document was rejected or an input/output error occurred.

Try `go run . -input tampered.pdf > rejected-report.json` to capture rejection. `-roots certs/root.pem` sets the explicit trust pool. An unsigned or missing file produces `valid: false`. This compact report avoids raw certificate objects and private data.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers JSON validity, acceptance and rejection, missing inputs and roots, and writer failures.

The runtime trust root is `certs/root.pem`; no PEM files are stored in this example folder.
