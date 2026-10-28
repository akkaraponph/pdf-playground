// Package testutil provides ephemeral fixtures for white-box tests only.
package testutil

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digitorus/timestamp"
	"pdf-playground/internal/demo"
)

// Fixture creates fresh signed, tampered and unsigned files with a separate root.
func Fixture(t *testing.T, n int) (string, string) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "certs", "root.pem")
	if err := demo.GenerateExample(dir, root, n); err != nil {
		t.Fatal(err)
	}
	return dir, root
}

// Credentials writes test-only PEM files into a temporary directory outside the repo.
func Credentials(t *testing.T) (*demo.Identity, string, string) {
	t.Helper()
	id, err := demo.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cert := filepath.Join(dir, "signer.crt.pem")
	key := filepath.Join(dir, "signer.key.pem")
	der, err := x509.MarshalPKCS8PrivateKey(id.Key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: id.Certificate.Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	return id, cert, key
}

// TSA serves real RFC 3161 responses locally; no external TSA is used by tests.
func TSA(t *testing.T) *httptest.Server {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(42), Subject: pkix.Name{CommonName: "Local test TSA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		req, err := timestamp.ParseRequest(body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		ts := &timestamp.Timestamp{HashAlgorithm: req.HashAlgorithm, HashedMessage: req.HashedMessage, Nonce: req.Nonce, Time: time.Now().UTC(), Policy: asn1.ObjectIdentifier{1, 2, 3, 4}, SerialNumber: big.NewInt(1), AddTSACertificate: true}
		response, err := ts.CreateResponseWithOpts(cert, key, crypto.SHA256)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(response)
	}))
	t.Cleanup(server.Close)
	return server
}

// FailingWriter exercises report serialization failures.
type FailingWriter struct{}

func (FailingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
