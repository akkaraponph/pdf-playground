# Example 3: Diagnose rejected signatures

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example3
go run .
```

Prints the self-signed signer, certificate errors, and timestamp warnings. Exit 0 means diagnosis completed, **not** that the PDF is accepted. Use `-input path.pdf` for another document.

The included timestamped PDF is labelled specifically for Example 3. The command uses `demo.Check` to collect per-signature errors; no roots are supplied, so trust fails. Timestamp presence and timestamp trust are shown separately.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers diagnostic fields and error output for signed, unsigned, and missing documents.
