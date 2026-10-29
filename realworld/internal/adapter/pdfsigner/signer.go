// Package pdfsigner implements real PAdES signing and offline verification.
package pdfsigner

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/digitorus/pdfsign"
	"pdf-playground/realworld/internal/document"
)

// Signer holds an explicitly configured signing identity and trust roots.
type Signer struct {
	key   crypto.Signer
	cert  *x509.Certificate
	roots *x509.CertPool
	chain []*x509.Certificate
}

var _ document.Signer = (*Signer)(nil)

func readBlock(path, kind string) (*pem.Block, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	if p == nil || p.Type != kind {
		return nil, fmt.Errorf("%s: expected PEM %s", path, kind)
	}
	return p, nil
}

// New loads a PKCS#8 private key, leaf certificate and trusted root bundle.
// The root bundle must build the leaf's chain; intermediates may be supplied
// after the leaf in certPath. No system trust store is used.
func New(certPath, keyPath, rootPath string) (*Signer, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	var certs []*x509.Certificate
	for len(bytes.TrimSpace(certPEM)) > 0 {
		block, rest := pem.Decode(certPEM)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("invalid certificate chain PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
		certPEM = rest
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("missing signer certificate")
	}
	block, err := readBlock(keyPath, "PRIVATE KEY")
	if err != nil {
		return nil, err
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("key must implement crypto.Signer")
	}
	pub, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return nil, err
	}
	certPub, err := x509.MarshalPKIXPublicKey(certs[0].PublicKey)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(pub, certPub) {
		return nil, fmt.Errorf("private key does not match signing certificate")
	}
	if certs[0].IsCA || certs[0].KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return nil, fmt.Errorf("leaf must permit digital signatures and must not be a CA")
	}
	rootsPEM, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootsPEM) {
		return nil, fmt.Errorf("no trusted root certificates")
	}
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	chains, err := certs[0].Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}})
	if err != nil {
		return nil, fmt.Errorf("signer chain: %w", err)
	}
	return &Signer{signer, certs[0], roots, chains[0][1:]}, nil
}

// Validate requires a parseable unsigned PDF; encrypted PDFs are not supported.
func (*Signer) Validate(data []byte) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("invalid PDF: %v", v)
		}
	}()
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return fmt.Errorf("missing PDF header")
	}
	doc, err := pdfsign.Open(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if doc.Reader().NumPage() < 1 {
		return fmt.Errorf("PDF has no pages")
	}
	for _, err := range doc.Signatures() {
		if err != nil {
			return err
		}
		return fmt.Errorf("PDF already signed")
	}
	return nil
}

// Sign produces a real PAdES B-B signature and verifies it before returning.
func (s *Signer) Sign(ctx context.Context, data []byte, reason string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	doc, err := pdfsign.Open(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	doc.Sign(s.key, s.cert, s.chain...).Format(pdfsign.PAdES_B_B).Reason(reason).Context(ctx)
	var out bytes.Buffer
	if _, err := doc.Write(&out); err != nil {
		return nil, err
	}
	report, err := s.Verify(ctx, out.Bytes())
	if err != nil {
		return nil, err
	}
	if !report.Valid || !report.Trusted {
		return nil, fmt.Errorf("signed output failed verification")
	}
	return out.Bytes(), nil
}

// Verify checks signature integrity and certificate trust offline. It does not
// assert revocation status or the trustworthiness of a timestamp authority.
func (s *Signer) Verify(ctx context.Context, data []byte) (report document.Verification, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("verify PDF: %v", v)
		}
	}()
	if err := ctx.Err(); err != nil {
		return report, err
	}
	doc, err := pdfsign.Open(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return report, err
	}
	result := doc.Verify().Strict().TrustedRoots(s.roots).ExternalChecks(false).Context(ctx)
	if err := result.Err(); err != nil {
		return report, err
	}
	report.Signatures = result.Count()
	report.Valid = report.Signatures > 0 && result.Valid()
	report.Trusted = report.Signatures > 0
	for _, signature := range result.Signatures() {
		report.Trusted = report.Trusted && signature.TrustedChain
	}
	return report, nil
}
