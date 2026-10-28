# Example 4: Detect changed signed content

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example4
go run .
```

The original signed PDF is accepted, then the amount changes from `100 USD` to `900 USD` and verification rejects `tampered.pdf`. Exit 0 means both expected outcomes occurred.

The source PDF deliberately uses uncompressed text. A same-length replacement preserves byte offsets and PDF syntax while changing signed bytes. This demonstrates modification of signed content, not every kind of permitted incremental PDF update.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers successful tampering detection, preservation of the original, invalid originals, missing roots, and output failures.

The runtime trust root is `certs/root.pem`; no PEM files are stored in this example folder.
