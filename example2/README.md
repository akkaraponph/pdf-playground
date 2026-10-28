# Example 2: Verify a signed PDF

This example uses `github.com/digitorus/pdfsign v1.0.0` from the repository root module and requires Go 1.27 or newer. It opens a PDF and uses `Verify().Strict()` to check its signatures. It also rejects documents without signatures.

```sh
cd example2
go run . -input contract-signed.pdf
```

The included PDF is generated specifically for Example 2 and uses a self-signed test certificate. Strict verification uses trusted roots and enforces certificate requirements, so this sample is expected to fail unless the signing certificate and timestamp chain satisfy the verifier's trust and strict policy. Use a signed PDF with an appropriately trusted certificate to test a successful verification. Strict mode also enables external checks, which can require network access.

Exit 1 is expected for the included PDF. Use `go run . -input /path/to/signed.pdf` for another file. Example5 demonstrates an explicit local CA trust setup; this example intentionally retains the basic `Strict()` flow.

## Expected Acrobat warning

The supplied PDF is a deliberate rejection case. Acrobat may report an unknown signer and an unverifiable timestamp while also reporting that the document has not been modified. This distinguishes intact signed content from a trusted signing identity; it is not the changed-content case shown in Example 4.

For a passing local trust example, run Example 5. Its root is supplied explicitly to Go and is not automatically trusted by Acrobat. Run `bash tools/retest-all.sh https://freetsa.org/tsr` from the repository root to regenerate and audit all example PDFs.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers strict rejection of untrusted, unsigned, modified, missing, and malformed PDFs.
