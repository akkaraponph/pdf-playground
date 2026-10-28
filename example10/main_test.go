package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"github.com/digitorus/pdfsign"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestNativeCertificateSigning(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	if err := run(dir, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "张伟") || !strings.Contains(out.String(), "测试签名有限公司") {
		t.Fatal(out.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "certs", "signer.crt.pem"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("missing PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "张伟" || cert.Subject.Organization[0] != "测试签名有限公司" || cert.Subject.Country[0] != "CN" {
		t.Fatal(cert.Subject)
	}
	roots, err := demo.Roots(filepath.Join(dir, "certs", "root.pem"))
	if err != nil {
		t.Fatal(err)
	}
	original := demo.Check(filepath.Join(dir, "signed.pdf"), roots)
	if !original.Valid || original.Signatures[0].Signer != "张伟" {
		t.Fatal(original)
	}
	if demo.Check(filepath.Join(dir, "signed.pdf"), x509.NewCertPool()).Valid {
		t.Fatal("untrusted signature accepted")
	}
	if demo.Check(filepath.Join(dir, "tampered.pdf"), roots).Valid {
		t.Fatal("modified content accepted")
	}
	doc, err := pdfsign.OpenFile(filepath.Join(dir, "signed.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	result := doc.Verify().Strict().TrustedRoots(roots).ExternalChecks(false)
	if result.Signatures()[0].Reason != "中文证书签名测试" {
		t.Fatalf("reason lost: %q", result.Signatures()[0].Reason)
	}
	key, err := os.ReadFile(filepath.Join(dir, "certs", "signer.key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if err := run(dir, &out); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "certs", "signer.key.pem"))
	if !bytes.Equal(key, after) {
		t.Fatal("rerun replaced identity")
	}
	if err := run(dir, testutil.FailingWriter{}); err == nil {
		t.Fatal("writer failure ignored")
	}
}
func TestInvalidOutputDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(path, &bytes.Buffer{}); err == nil {
		t.Fatal("file used as directory")
	}
}
