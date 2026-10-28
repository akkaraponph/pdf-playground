// Package demo supplies local-only fixtures for the blog examples.
package demo

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/digitorus/pdfsign"
)

// Identity holds ephemeral demo credentials. Keys are never written to disk.
type Identity struct {
	Key               crypto.Signer
	Certificate, Root *x509.Certificate
}

// NewIdentity creates a local test CA and a signing certificate with the usages
// required by Strict. Certificates have no network revocation endpoints.
func NewIdentity() (*Identity, error) {
	return NewIdentityForSubject(pkix.Name{CommonName: "PDF playground example signer"})
}

// NewIdentityForSubject issues a local test certificate preserving Unicode subject fields.
func NewIdentityForSubject(subject pkix.Name) (*Identity, error) {
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	rootTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "PDF playground test CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(10, 0, 0), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	der, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, err
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	leaf := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: subject, NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(1, 0, 0), BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}}
	der, err = x509.CreateCertificate(rand.Reader, leaf, root, &key.PublicKey, rootKey)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &Identity{key, cert, root}, nil
}

// PDF returns a small uncompressed one-page PDF so the tamper example can change
// visible content without changing byte offsets.
func PDF() []byte {
	return PDFFor(0, "Signing test", "Use the verification command to check this document")
}

// PDFFor gives every example and case a visible label before signing.
func PDFFor(example int, caseName, expected string) []byte {
	titles := map[int]string{1: "Sign a PDF using PAdES B-T", 2: "Strict signature verification", 3: "Diagnose verification failures", 4: "Detect changed signed content", 5: "Trust a local test CA", 6: "Compare B-B and B-T timestamps", 7: "JSON verification report", 8: "Batch PDF verification", 9: "Thai certificate signing", 10: "Chinese certificate signing"}
	title := fmt.Sprintf("Example %d - %s", example, titles[example])
	lines := []string{title, "github.com/digitorus/pdfsign v1.0.0", "", "Case: " + caseName, "Expected result: " + expected, "", "Case state: ORIGINAL", "Contract amount: 100 USD", "", "Learning fixture - generated locally with test credentials.", "This label describes the case; it is not proof of verification.", "Run the Go command in this example folder to check the PDF."}
	var stream strings.Builder
	stream.WriteString("BT /F1 16 Tf 54 738 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -26 Td /F1 11 Tf\n")
		}
		escaped := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(line)
		fmt.Fprintf(&stream, "(%s) Tj\n", escaped)
	}
	stream.WriteString("ET")
	content := []byte(stream.String())
	objects := [][]byte{[]byte("<< /Type /Catalog /Pages 2 0 R >>"), []byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"), []byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>"), []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"), []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content))}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return b.Bytes()
}

// Sign creates a B-B signature, or B-T if an explicit TSA URL is supplied.
func Sign(input, output string, id *Identity, tsa string) error {
	doc, err := pdfsign.OpenFile(input)
	if err != nil {
		return err
	}
	var chain []*x509.Certificate
	if id.Root != nil && !bytes.Equal(id.Certificate.Raw, id.Root.Raw) {
		chain = append(chain, id.Root)
	}
	builder := doc.Sign(id.Key, id.Certificate, chain...).Format(pdfsign.PAdES_B_B).Reason("Playground contract approval")
	if tsa != "" {
		builder.Format(pdfsign.PAdES_B_T).Timestamp(tsa)
	}
	var buf bytes.Buffer
	if _, err := doc.Write(&buf); err != nil {
		return err
	}
	return os.WriteFile(output, buf.Bytes(), 0644)
}

// Generate creates unsigned, signed, and changed-content PDFs plus a public root.
func Generate(dir string) error { return GenerateExample(dir, filepath.Join(dir, "root.pem"), 4) }

// GenerateExample creates case-labelled fixtures for one example.
func GenerateExample(dir, rootPath string, example int) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	id, err := NewIdentity()
	if err != nil {
		return err
	}
	input := filepath.Join(dir, "unsigned.pdf")
	signed := filepath.Join(dir, "signed.pdf")
	if err := os.WriteFile(input, PDFFor(example, "Unsigned document", "REJECTED - no signatures"), 0644); err != nil {
		return err
	}
	source := filepath.Join(dir, "signing-source.pdf")
	if err := os.WriteFile(source, PDFFor(example, "Signed document - original content", "ACCEPTED with the matching test CA"), 0644); err != nil {
		return err
	}
	defer os.Remove(source)
	if err := Sign(source, signed, id, ""); err != nil {
		return err
	}
	data, err := os.ReadFile(signed)
	if err != nil {
		return err
	}
	if bytes.Count(data, []byte("100 USD")) != 1 {
		return fmt.Errorf("expected one amount marker")
	}
	data, err = Tamper(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tampered.pdf"), data, 0644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(rootPath), 0700); err != nil {
		return err
	}
	return os.WriteFile(rootPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: id.Root.Raw}), 0644)
}

// Tamper changes visible text within the signed bytes, keeping lengths stable.
func Tamper(data []byte) ([]byte, error) {
	for _, change := range [][2]string{{"Signed document - original content", "Signed document - modified content"}, {"100 USD", "900 USD"}, {"Case state: ORIGINAL", "Case state: MODIFIED"}, {"ACCEPTED with the matching test CA", "REJECTED with the matching test CA"}} {
		if bytes.Count(data, []byte(change[0])) != 1 {
			return nil, fmt.Errorf("expected one marker %q", change[0])
		}
		data = bytes.Replace(data, []byte(change[0]), []byte(change[1]), 1)
	}
	return data, nil
}
