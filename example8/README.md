# Example 8: Verify a batch of PDFs

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example8
go run . > batch-report.json
```

The default batch accepts one signed PDF and rejects the modified and unsigned PDFs. **Exit 1 is expected**, while the JSON report is still written.

Pass PDF paths as arguments: `go run . signed.pdf missing.pdf`. Each document gets its own result; a missing file does not stop processing the others. Exit 0 requires every document to be accepted. Invalid trust-root configuration fails the whole command.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers all-accepted and mixed batches, processing after individual failures, empty input, missing roots, and writer failures.

The runtime trust root is `certs/root.pem`; no PEM files are stored in this example folder.
