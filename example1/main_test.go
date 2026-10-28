package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestSignPDFRejectsSameFile(t *testing.T) {
	id, err := demo.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"relative-alias", "hard-link"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "source.pdf")
			original := demo.PDF()
			if err := os.WriteFile(input, original, 0600); err != nil {
				t.Fatal(err)
			}
			output := dir + "/./source.pdf"
			if kind == "hard-link" {
				output = filepath.Join(dir, "link.pdf")
				if err := os.Link(input, output); err != nil {
					t.Fatal(err)
				}
			}
			if err := signPDF(input, output, id.Key, id.Certificate, "http://127.0.0.1:1"); err == nil {
				t.Fatal("same file accepted")
			}
			data, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(data, original) {
				t.Fatal("source PDF overwritten through path alias")
			}
		})
	}
}

func TestRunLoadsKeysAndSigns(t *testing.T) {
	id, cert, key := testutil.Credentials(t)
	tsa := testutil.TSA(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pdf")
	if err := os.WriteFile(input, demo.PDF(), 0600); err != nil {
		t.Fatal(err)
	}
	rsaKey := id.Key.(*rsa.PrivateKey)
	for _, tc := range []struct {
		name string
		der  []byte
		kind string
	}{{"pkcs8", nil, ""}, {"pkcs1", x509.MarshalPKCS1PrivateKey(rsaKey), "RSA PRIVATE KEY"}} {
		t.Run(tc.name, func(t *testing.T) {
			keyPath := key
			if tc.der != nil {
				keyPath = filepath.Join(dir, "rsa.key.pem")
				if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: tc.kind, Bytes: tc.der}), 0600); err != nil {
					t.Fatal(err)
				}
			}
			output := filepath.Join(dir, tc.name+".pdf")
			if err := run(input, output, cert, keyPath, tsa.URL); err != nil {
				t.Fatal(err)
			}
			pool := x509.NewCertPool()
			pool.AddCert(id.Root)
			r := demo.Check(output, pool)
			if !r.Valid || len(r.Signatures) != 1 || !r.Signatures[0].TimestampPresent {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestCredentialErrors(t *testing.T) {
	_, cert, key := testutil.Credentials(t)
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	badDER := filepath.Join(dir, "bad-der.pem")
	if err := os.WriteFile(badDER, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid DER")}), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ cert, key, contains string }{{"missing.pem", key, "read certificate"}, {bad, key, "expected a PEM certificate"}, {badDER, key, "parse certificate"}, {cert, "missing.pem", "read private key"}, {cert, bad, "expected an unencrypted PEM private key"}, {cert, badDER, "unsupported private key"}} {
		if err := run("unused.pdf", "unused-out.pdf", tc.cert, tc.key, "unused"); err == nil || !strings.Contains(err.Error(), tc.contains) {
			t.Fatalf("want %s got %v", tc.contains, err)
		}
	}
	for _, kind := range []string{"PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY"} {
		badKey := filepath.Join(dir, "invalid.key.pem")
		if err := os.WriteFile(badKey, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: []byte("bad DER")}), 0600); err != nil {
			t.Fatal(err)
		}
		if err := run("unused.pdf", "unused-out.pdf", cert, badKey, "unused"); err == nil || !strings.Contains(err.Error(), "parse private key") {
			t.Fatal(err)
		}
	}
}

func TestTimestampFailurePreservesOutput(t *testing.T) {
	id, _, _ := testutil.Credentials(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pdf")
	output := filepath.Join(dir, "output.pdf")
	if err := os.WriteFile(input, demo.PDF(), 0600); err != nil {
		t.Fatal(err)
	}
	original := []byte("existing output must survive failure")
	if err := os.WriteFile(output, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := signPDF(input, output, id.Key, id.Certificate, "http://127.0.0.1:1"); err == nil {
		t.Fatal("TSA failure accepted")
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, original) {
		t.Fatal("failed TSA request destroyed existing output")
	}
}
