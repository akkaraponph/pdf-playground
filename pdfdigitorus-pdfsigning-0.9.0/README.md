# digitorus/pdfsign v0.9.0 signing sample

This is a small, runnable integration test for the released
`github.com/digitorus/pdfsign v0.9.0` API. It uses the v0.9.0 packages:

- `github.com/digitorus/pdfsign/sign.SignFile`
- `github.com/digitorus/pdfsign/verify.VerifyWithOptions`

## Test the signing result

```sh
cd pdfdigitorus-pdfsigning-0.9.0
go test ./...
```

The test creates a one-page PDF, generates an ephemeral self-signed RSA
certificate, signs the PDF, and checks that v0.9.0 reports one valid signature.
The certificate is created only in memory and is not written to the repository.

## Run the sample commands

Generate a mock PDF:

```sh
go run ./cmd/generate-mock-pdf
```

Sign a PDF with a PEM certificate and RSA private key (PKCS#1 or PKCS#8):

```sh
go run ./cmd/sign-pdf \
  -input testdata/mock-document.pdf \
  -output results/mock-document-signed.pdf \
  -certificate /secure/path/signer.crt \
  -private-key /secure/path/signer.key
```

If local ignored test credentials are available under `certs/`, the same
sample can be run with:

```sh
go run ./cmd/sign-pdf \
  -input testdata/mock-document.pdf \
  -output results/mock-document-signed.pdf \
  -certificate certs/akkarapon-phikulsri-testing.crt \
  -private-key certs/akkarapon-phikulsri-testing.key
```

The command also verifies the resulting PDF and prints the signer name. Keep
private keys, passwords, and certificate bundles outside this repository.

This sample enables `AllowUntrustedRoots` only for local self-signed testing.
Production verification should use a configured trust store instead.
