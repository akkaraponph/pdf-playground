# Signing test

This directory is a small executable use case for testing
[`digitorus/pdfsign` PR #169](https://github.com/digitorus/pdfsign/pull/169).
It generates a one-page PDF, applies a `CertificationSignature`, and verifies
the resulting file.

## Test purpose

PR #169 makes a certification signature write the catalog's `/Perms /DocMDP`
entry. That entry connects the catalog to the signature dictionary's DocMDP
transform, so compatible PDF readers can enforce certification permissions.

The test also provides a reproducible input for checking that the output is
cryptographically valid and that the certification metadata is present.

## Requirements

- Go 1.27 or newer.
- A signing certificate and matching PEM private key supplied outside this
  repository.
- The local `github.com/digitorus/pdfsign` checkout configured by `go.mod` must
  contain the PR #169 changes.

The command does not have certificate defaults. This prevents an old or
unintended certificate from being used accidentally.

## Quick start

```sh
cd signingtest
go test ./...
go run ./cmd/generate-mock-pdf
go run ./cmd/sign-pdf \
  -input testdata/mock-signing-document.pdf \
  -output results/mock-signing-document-signed.pdf \
  -certificate /secure/path/signer.crt \
  -private-key /secure/path/signer.key
```

Add `-chain /secure/path/chain.crt` when intermediate or root certificates
should be embedded in the PDF.

## Expected result

The command should print a successful signing and verification message, and
create:

```text
results/mock-signing-document-signed.pdf
```

Inspect the signature with Poppler when available:

```sh
pdfsig results/mock-signing-document-signed.pdf
```

The signature should be reported as valid and cover the complete document.
For the PR-specific catalog check:

```sh
qpdf --qdf --object-streams=disable \
  results/mock-signing-document-signed.pdf \
  /tmp/mock-signing-document-signed.qdf.pdf
rg -a -n "/Perms|/DocMDP|/TransformMethod|/P " \
  /tmp/mock-signing-document-signed.qdf.pdf
```

The catalog must contain `/Perms` with `/DocMDP` pointing to the signature
dictionary. The signature dictionary must contain `/TransformMethod /DocMDP`
and the configured permission `/P`.

## Test matrix

| Scenario | Expected result |
| --- | --- |
| Unsigned PDF + `CertificationSignature` | Signs successfully; catalog gets `/Perms /DocMDP`. |
| Already certified PDF + `CertificationSignature` | Rejected before writing. |
| PDF with an already signed field + `CertificationSignature` | Rejected before writing. |
| Existing `/Perms` entries | Preserved while `/DocMDP` is merged. |
| Escaped catalog names/literal strings | Round-trip without changing catalog structure. |

The focused implementation tests live in the `pdfsign` checkout. Run them
there for the full PR coverage:

```sh
cd /path/to/pdfsign
go test ./sign -run 'Test(CreateCatalog|.*DocMDP.*)' -count=1
go test ./...
```

## Certificate handling

Do not commit `.key`, `.p12`, `.pfx`, passphrase, or base64 certificate files.
Use temporary or externally managed paths and pass them with `-certificate`,
`-private-key`, and optionally `-chain`.

For details and troubleshooting, see [`../howto/README.md`](../howto/README.md).
