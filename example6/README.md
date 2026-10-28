# Example 6: Compare PAdES B-B and B-T timestamps

Uses `github.com/digitorus/pdfsign v1.0.0` from the repository root Go module. Requires Go 1.27 or newer.

```sh
cd example6
go run .
```

`baseline.pdf` has no timestamp; `timestamped.pdf` has one. Timestamp trust is reported separately and is false for the included historical FreeTSA sample.

B-B is generated offline using a local test CA. B-T uses an untrusted self-signed test certificate, so the two fixtures use different signers; this is a profile demonstration, not a controlled comparison of certificate trust. To create a fresh B-T PDF with an ephemeral local signer, run `go run . -tsa https://freetsa.org/tsr`. It writes `timestamped-live.pdf`. A TSA error is returned; the command never falls back to B-B.

Verification in examples 3–8 disables external revocation fetching for reproducible offline demonstrations. Passing these checks does not establish current revocation status or legal qualification. The test certificates expire; regenerate local fixtures from the repository root with `go run ./tools/generate-fixtures`. This replaces the local test CA and related fixtures together; keys exist only in memory.

## White-box tests

Run from this example folder:

```sh
go test -count=1 -cover .
```

Tests call the internal functions directly using fresh temporary PDFs and certificates. They do not need the stored test credentials or an external TSA.
Covers timestamp presence, a local RFC 3161 signing response, unreachable TSA failures, and unsigned or missing comparison inputs.

The runtime trust root is `certs/root.pem`; no PEM files are stored in this example folder.
