package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"

	pdfsigning "pdfdigitorus-pdfsigning-0.9.0"
)

func main() {
	input := flag.String("input", "testdata/mock-document.pdf", "input PDF path")
	output := flag.String("output", "results/mock-document-signed.pdf", "signed PDF path")
	certificatePath := flag.String("certificate", "", "PEM certificate path")
	privateKeyPath := flag.String("private-key", "", "PEM RSA private key path")
	flag.Parse()

	if *certificatePath == "" || *privateKeyPath == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "-certificate and -private-key are required")
		os.Exit(2)
	}

	certificate, err := loadCertificate(*certificatePath)
	if err != nil {
		fatal("load certificate", err)
	}
	signer, err := loadPrivateKey(*privateKeyPath)
	if err != nil {
		fatal("load private key", err)
	}
	if err := pdfsigning.SignPDF(*input, *output, certificate, signer); err != nil {
		fatal("sign PDF", err)
	}

	response, err := pdfsigning.VerifyPDF(*output)
	if err != nil {
		fatal("verify PDF", err)
	}
	if len(response.Signers) != 1 || !response.Signers[0].ValidSignature {
		fatal("verify PDF", fmt.Errorf("expected one valid signature, got %d", len(response.Signers)))
	}
	fmt.Printf("signed and verified %s as %s\n", *output, response.Signers[0].Name)
}

func loadCertificate(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("no PEM certificate found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func loadPrivateKey(path string) (crypto.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM private key found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if signer, ok := key.(crypto.Signer); ok {
			return signer, nil
		}
		return nil, fmt.Errorf("PKCS#8 key does not implement crypto.Signer")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RSA private key: %w", err)
	}
	return (*rsa.PrivateKey)(key), nil
}

func fatal(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}
