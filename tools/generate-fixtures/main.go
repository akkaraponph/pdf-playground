// Run from the repository root. Use -tsa URL to also regenerate B-T samples.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
	"time"
)

func main() {
	tsa := flag.String("tsa", "", "optional RFC 3161 URL for B-T samples")
	flag.Parse()
	if err := run(*tsa); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(tsa string) error {
	if _, err := os.Stat("example1/main.go"); err != nil {
		return fmt.Errorf("run from repo root: %w", err)
	}
	for n := 4; n <= 8; n++ {
		if err := demo.GenerateExample(fmt.Sprintf("example%d", n), fmt.Sprintf("example%d/certs/root.pem", n), n); err != nil {
			return err
		}
	}
	if err := os.WriteFile("example1/contract.pdf", demo.PDFFor(1, "PAdES B-T signing", "Timestamp present after signing with a reachable TSA"), 0644); err != nil {
		return err
	}
	// B-B needs no TSA and remains reproducible offline.
	id, err := demo.NewIdentity()
	if err != nil {
		return err
	}
	if err := signCase(6, "B-B signature without timestamp", "No signature timestamp", "example6/baseline.pdf", id, ""); err != nil {
		return err
	}
	if tsa != "" {
		untrusted, err := selfSigned()
		if err != nil {
			return err
		}
		if err := demo.Sign("example1/contract.pdf", "example1/contract-signed.pdf", untrusted, tsa); err != nil {
			return err
		}
		for _, n := range []int{2, 3} {
			if err := signCase(n, "Untrusted test signer with a timestamp", "REJECTED by strict verification without trusted roots", fmt.Sprintf("example%d/contract-signed.pdf", n), untrusted, tsa); err != nil {
				return err
			}
		}
		if err := signCase(6, "B-T signature with a timestamp", "Timestamp present; TSA trust must be checked separately", "example6/timestamped.pdf", untrusted, tsa); err != nil {
			return err
		}
	}
	fmt.Println("Generated labelled v1.0.0 fixtures; private keys were kept in memory")
	if tsa == "" {
		fmt.Println("Existing B-T results preserved; use -tsa URL to regenerate them")
	}
	return nil
}
func signCase(n int, name, expected, output string, id *demo.Identity, tsa string) error {
	dir, err := os.MkdirTemp("", "pdf-case-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "source.pdf")
	if err := os.WriteFile(input, demo.PDFFor(n, name, expected), 0600); err != nil {
		return err
	}
	return demo.Sign(input, output, id, tsa)
}
func selfSigned() (*demo.Identity, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(100), Subject: pkix.Name{CommonName: "Example PDF Signer"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(1, 0, 0)}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &demo.Identity{Key: key, Certificate: parsed, Root: parsed}, nil
}
