# Example 9: Thai certificate signing

Uses `github.com/digitorus/pdfsign v1.0.0` from the root module, requiring Go 1.27 or newer. This example tests Unicode certificate subjects and signature metadata with PAdES B-B. No TSA or network connection is required.

```sh
cd example9
go run .
```

The test subject is:

| Field | Value |
| --- | --- |
| Common name | สมชาย ใจดี |
| Organization | บริษัท ทดสอบลายมือชื่อ จำกัด |
| Country | TH |
| Signature reason | ทดสอบการลงนามเอกสารภาษาไทย |

The command creates a local test CA and RSA 2048 signing identity on the first run. Credentials live in `certs/signer.key.pem`, `certs/signer.crt.pem`, and `certs/root.pem`, ignored by `certs/.gitignore`. The private key is PKCS#8 with permissions `0600`. Existing matching credentials are reused. Partial, corrupt, or differently named credentials are rejected without being replaced.

Outputs:

- `signed.pdf`: expected to pass strict offline verification with `certs/root.pem`; the embedded signer certificate and signature metadata preserve the native name.
- `tampered.pdf`: signed amount and case labels are changed; expected to fail integrity verification.
- `unsigned.pdf`: expected to be rejected because it has no signature.
- Console output: native certificate subject, signer, reason, and verification result.

The page labels are English. The language test is in the embedded X.509 certificate and PDF signature metadata; it does not test Thai/Chinese page-text fonts or shaping. Inspect certificate details and the signature panel in a PDF viewer. Acrobat has a separate trust store and may warn that the test signer is unknown even when Go verification passes with the explicitly supplied root.

## White-box tests

```sh
go test -count=1 -cover .
```

Tests invoke `run` directly with temporary directories. They check native subject fields after parsing the actual PEM certificate, successful signing and trust, Unicode metadata readback, rejection without a trusted root, modified-content rejection, credential reuse, invalid output paths, and writer failures. Shared tests additionally cover incomplete and corrupt credentials. Test keys and certificates are temporary and do not depend on this folder's stored credentials.

To regenerate all ten examples and audit every PDF from the repository root:

```sh
bash tools/retest-all.sh https://freetsa.org/tsr
```
