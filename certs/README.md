# Local mock signing certificates

These credentials are generated for playground testing only. All generated credential files and subfolders are ignored by Git; only this README and `.gitignore` are intended for tracking.

- `signer.key.pem`: unencrypted RSA 2048 private key in PKCS#8 format; permissions 0600.
- `signer.crt.pem`: signing certificate, valid for one year from generation.
- `root.crt.pem`: public local test CA, valid for ten years.
- `chain.pem`: signing certificate followed by the public test root.

The signer permits digital signatures and content commitment and has email-protection EKU, which pdfsign v1.0.0 accepts. This local CA is not publicly trusted. Keep its trust limited to an explicit certificate pool in your test application.

Generate from the repository root:

```sh
go run ./tools/generate-certs
```

The generator refuses to overwrite existing credentials. Choose another directory with `-output /path/to/new-test-certs` to create another identity.

Sign using example1:

```sh
cd example1
go run . -certificate ../certs/signer.crt.pem \
  -private-key ../certs/signer.key.pem -tsa https://freetsa.org/tsr \
  -output contract-mock-signed.pdf
```

Inspect the chain with OpenSSL:

```sh
openssl verify -CAfile certs/root.crt.pem certs/signer.crt.pem
```

The fixture roots inside examples4–8 belong to separately generated identities. Use `certs/root.crt.pem` when verifying PDFs signed with these credentials. Timestamp trust is checked separately from signer trust.

## Per-example fixture roots

`go run ./tools/generate-fixtures` writes public test roots to `example4/certs/root.pem` through `example8/certs/root.pem`, alongside their matching signed fixtures. Each example has a `certs/.gitignore` that ignores its generated root. The example commands load `certs/root.pem` relative to their own folder; no PEM files sit beside `main.go`. Regenerating fixtures replaces these roots together with the PDFs they verify. The reusable mock signing credentials above are preserved.

Run `python3 tools/check-credentials.py` from the repository root to check that all credentials are in a `certs/` folder, ignored, and absent from the Git index.
