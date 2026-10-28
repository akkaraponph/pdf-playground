package pdfsigning

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMockPDFSigningProducesValidV090Signature(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	inputPath := filepath.Join(directory, "mock-document.pdf")
	outputPath := filepath.Join(directory, "mock-document-signed.pdf")

	if err := GenerateMockPDF(inputPath); err != nil {
		t.Fatalf("generate mock PDF: %v", err)
	}

	certificate, privateKey := testCertificate(t)
	if err := SignPDF(inputPath, outputPath, certificate, privateKey); err != nil {
		t.Fatalf("sign PDF: %v", err)
	}

	response, err := VerifyPDF(outputPath)
	if err != nil {
		t.Fatalf("verify signed PDF: %v", err)
	}
	if len(response.Signers) != 1 {
		t.Fatalf("expected one signer, got %d", len(response.Signers))
	}
	if !response.Signers[0].ValidSignature {
		t.Fatalf("expected a valid cryptographic signature; verification error: %q", response.Error)
	}
	if response.Signers[0].Name != certificate.Subject.CommonName {
		t.Fatalf("expected signer %q, got %q", certificate.Subject.CommonName, response.Signers[0].Name)
	}
	if response.DocumentInfo.Pages != 1 {
		t.Fatalf("expected one page, got %d", response.DocumentInfo.Pages)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("signed PDF was not written: %v", err)
	}
}

func testCertificate(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "pdfsign v0.9.0 mock signer"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return certificate, privateKey
}
