# Example 5: Verify using an explicit local test CA

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example5
go run .
```

Prints that the PDF passed strict certificate checks using the playground root.

The visible fluent API uses `Verify().Strict().TrustedRoots(roots).ExternalChecks(false)`. The generated signer has digital-signature and content-commitment key usages plus email-protection EKU accepted by this library. `certs/root.pem` is loaded only into a local certificate pool. No trust bypass is enabled.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers the matching root, a wrong root, missing roots, and signed, unsigned, modified, and missing documents.

The runtime trust root is `certs/root.pem`; no PEM files are stored in this example folder.
