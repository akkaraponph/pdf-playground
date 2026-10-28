# digitorus/pdfsign v1.0.0: PAdES B-T example

This example uses the repository root Go module to sign a PDF with the v1.0.0 fluent API and requests an RFC 3161 signature timestamp. Go 1.27 or newer is required by pdfsign.

```sh
cd example1
go run . -input contract.pdf -output contract-signed.pdf \
  -certificate /secure/path/signer.crt \
  -private-key /secure/path/signer.key \
  -tsa https://your-timestamp-authority.example/tsr
```

Supply a matching PEM certificate and unencrypted PKCS#8, RSA PKCS#1, or EC private key, and a reachable TSA endpoint. Keep credentials outside the repository. The output path should be different from the input; an existing output file is overwritten and signing completes in memory before the output is written, so a TSA failure preserves existing output.

The example opens the PDF, configures `Sign`, `Format(PAdES_B_T)`, `Timestamp`, and `Reason`, and writes the result. It does not implement credential security, input validation, certificate or TSA trust policies, or signature verification; production applications must address these separately.

The included `contract.pdf` is a sample input. `contract-signed.pdf` is an example output signed with a temporary self-signed test certificate and a FreeTSA timestamp; it is for demonstration, not a trusted production signature.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers credential parsing, key formats, real signing with a local RFC 3161 test TSA, input/output aliases, and preserving output after TSA failure.
