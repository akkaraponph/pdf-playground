# PDF signing playground — pdfsign v1.0.0

Ten Go examples for learning and blog demonstrations, using `github.com/digitorus/pdfsign v1.0.0`. Requires Go 1.27 or newer. Run commands from the indicated example folder.

- [Example 1: Sign using PAdES B-T](example1/README.md)
- [Example 2: Strict verification](example2/README.md)
- [Example 3: Diagnose verification failures](example3/README.md)
- [Example 4: Detect changed signed content](example4/README.md)
- [Example 5: Trust a local test CA](example5/README.md)
- [Example 6: Compare B-B and B-T timestamps](example6/README.md)
- [Example 7: Produce a JSON verification report](example7/README.md)
- [Example 8: Verify a batch of PDFs](example8/README.md)
- [Example 9: Thai certificate signing](example9/README.md)
- [Example 10: Chinese certificate signing](example10/README.md)

The root `go.mod` pins v1.0.0 for all ten examples. To add that exact release to another module, use `go get github.com/digitorus/pdfsign@v1.0.0`.

Each folder contains sample PDFs and explains expected behavior. A command that demonstrates rejection is not a broken example. Signature integrity, certificate trust, and timestamp trust are separate checks.

Examples 4–8 use a generated local test CA where appropriate. This CA is only for the playground; never install it in your system trust store. Fixture private keys exist only in memory. Reusable mock credentials can be generated separately under `certs/` as described below; their PEM files are ignored by Git. Example1's timestamped PDF was created with a temporary self-signed certificate and FreeTSA; strict verification rejects that sample.

## Generate certificates for testing

Use the included [certificate-generation script](tools/generate-certs/main.go). From the repository root, with Go 1.27 or newer installed, run:

```sh
go run ./tools/generate-certs
```

This creates a local test CA and a matching RSA 2048 signing identity in `certs/`:

- `certs/signer.key.pem`: unencrypted PKCS#8 private key with permissions `0600`.
- `certs/signer.crt.pem`: signing certificate, valid for one year.
- `certs/root.crt.pem`: public test CA certificate, valid for ten years.
- `certs/chain.pem`: signing certificate followed by the test CA certificate.

The generator refuses to overwrite existing credentials. To generate a separate identity, choose a new directory:

```sh
go run ./tools/generate-certs -output /tmp/pdfsign-test-certs
```

All generated credentials under `certs/`, including per-example roots in `exampleN/certs/`, are ignored by Git. These credentials are for local testing; trust this CA explicitly in your application rather than installing it into the system trust store. Certificate and key file extensions are also ignored repository-wide as a safeguard; keep generated credentials under `certs/`.

Check the certificate chain using OpenSSL, from the repository root:

```sh
openssl verify -CAfile certs/root.crt.pem certs/signer.crt.pem
```

Expected output: `certs/signer.crt.pem: OK`.

Sign the Example 1 input using the generated credentials, also from the repository root:

```sh
go run ./example1 \
  -input example1/contract.pdf \
  -output example1/contract-mock-signed.pdf \
  -certificate certs/signer.crt.pem \
  -private-key certs/signer.key.pem \
  -tsa https://freetsa.org/tsr
```

This writes `example1/contract-mock-signed.pdf` and requires access to the TSA. Use `certs/root.crt.pem` as the explicit signer trust root when verifying PDFs signed with these credentials. The fixture roots in examples 4–8 are separate identities, and timestamp trust is checked separately. See [certs/README.md](certs/README.md) for more details.

## Reproduce and check

The fixture generator creates local PDF samples, a public test root certificate, and signed results. Offline examples require no TSA. Live timestamp creation requires a reachable RFC 3161 endpoint.

```sh
go run ./tools/generate-fixtures
go test ./...
go vet ./...
python3 tools/check-examples.py
```

The older experiments under `fix-issues/` are historical and use their own Go modules; they are outside this ten-example series.

## Inspect PDF permissions

```sh
qpdf --qdf --object-streams=disable example1/contract-signed.pdf unpacked.pdf
rg '/Perms|/DocMDP|/UR3' unpacked.pdf
```

## Read the case inside each PDF

Every PDF displays its example number, topic, case, and expected outcome. Signed documents receive their labels **before** signing. Modified fixtures change both the amount and the case state from ORIGINAL to MODIFIED, so they remain readable while verification rejects their altered signed bytes. A label is a description, not proof of verification.

Regenerate offline fixtures as above. To also rebuild each example's timestamped PDFs with its own labels:

```sh
go run ./tools/generate-fixtures -tsa https://freetsa.org/tsr
cd example6
go run . -tsa https://freetsa.org/tsr
```


## Retest every generated PDF

Install Go 1.27+, Python 3, qpdf and Poppler (`pdfsig` and `pdftotext`). Then run from the repository root:

```sh
bash tools/retest-all.sh https://freetsa.org/tsr
```

This regenerates **all** PDFs, including live timestamp signatures and the mock-certificate output; refreshes saved command results; runs tests and vet; and audits every PDF using pdfsign plus Poppler's independent signature-integrity check. Existing mock credentials are preserved. The command needs network access to the chosen TSA and exits on generation or audit failures. The report is saved to [docs/pdf-audit.json](docs/pdf-audit.json), with a hash identifying each tested PDF.

To audit the current files without regenerating them:

```sh
python3 tools/audit-pdfs.py
```

### What to expect in Adobe Acrobat

Example 2 intentionally contains an **untrusted test signer**. Acrobat may show “At least one signature has problems”, alongside “Document has not been modified” and “Signer's identity is unknown”. That matches this rejection case: the signed bytes are intact, but the certificate is not trusted. Its embedded timestamp also requires separate TSA validation.

- **Examples 1–3 test signatures:** signed content is intact, but the signer is untrusted, so strict verification rejects the test identity.
- **`signed.pdf` in examples 4–10:** Go verification passes with the example's explicit `certs/root.pem`. Acrobat may still warn because it has a separate trust store.
- **`tampered.pdf` in examples 4–10:** verification detects a digest mismatch because signed bytes changed.
- **`unsigned.pdf` and the Example 1 input:** these contain no signature and are rejected.
- **Example 6 timestamp cases:** timestamp presence is reported separately from timestamp trust.
- **Example 1 mock certificate output:** the signer is trusted when `certs/root.crt.pem` is supplied; timestamp trust is a separate check.

A passing audit means every file matches its expected case, including intentional failures. It does **not** mean all PDFs have trusted signatures. The audit does not automate Acrobat, modify its trust store, or prove current revocation status. In pdfsign v1.0.0, `Valid()` can be true while `TimestampValid` is false and a timestamp warning is present; inspect both fields when timestamp trust is required. See [verification notes](docs/verification-results.md) for the retest findings.

## White-box testing and credential layout

Each `exampleN/main_test.go` uses `package main` to test the example's own functions, including success and failure branches. Tests generate their credentials in temporary directories and serve timestamp responses on a local test HTTP server; they do not contact a public TSA. The live regeneration script separately exercises the configured external TSA.

```sh
go test -count=1 -cover ./...
go test -race ./...
python3 tools/check-credentials.py
```

Runtime credentials belong in ignored `certs/` folders, each containing its own `.gitignore`:

```text
certs/
  .gitignore             # Ignore credentials; keep this file and README
  README.md
  signer.key.pem         # Reusable mock key; ignored
  signer.crt.pem         # Mock signing certificate; ignored
  root.crt.pem           # Mock CA; ignored
  chain.pem              # Mock chain; ignored
example4/certs/
  .gitignore
  root.pem               # Fixture trust root; ignored
example5/certs/           # Same layout for examples 5–8
example6/certs/
example7/certs/
example8/certs/
example9/certs/           # Thai test identity and root; ignored
example10/certs/          # Chinese test identity and root; ignored
```

No standalone PEM, key, or certificate bundle should be committed. `tools/check-credentials.py` rejects credential files in Git's index, files outside a `certs/` folder, unignored credentials, and private-key PEM blocks outside that folder. Signed PDFs necessarily embed the public signing certificates; they contain no private keys. After a fresh checkout, run `go run ./tools/generate-certs` and regenerate fixtures before running commands that use these local roots. Run the full `tools/retest-all.sh` command above to rebuild and verify the entire series.

## Thai and Chinese certificate tests

[Example 9](example9/README.md) uses `CN=สมชาย ใจดี`, organization `บริษัท ทดสอบลายมือชื่อ จำกัด`, country `TH`. [Example 10](example10/README.md) uses `CN=张伟`, organization `测试签名有限公司`, country `CN`. Both also put native text in the PDF signature metadata, then verify the signed document using the local test root. They use PAdES B-B so these language tests work offline.

```sh
(cd example9 && go run . && go test -count=1 -cover .)
(cd example10 && go run . && go test -count=1 -cover .)
```

The first run generates `certs/signer.key.pem`, `certs/signer.crt.pem`, and `certs/root.pem` inside each example. Later runs reuse the identity. Each `certs/.gitignore` ignores those generated files; partial or mismatched credentials cause an error instead of silent replacement. The signed PDF embeds the public certificate, while the private key stays ignored. Each example also creates unsigned and modified negative cases.
