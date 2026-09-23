package main

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/digitorus/pdfsign"
)

func main() {
	input := flag.String("input", "testdata/mock-signing-document.pdf", "PDF to sign")
	output := flag.String("output", "results/mock-signing-document-signed.pdf", "signed PDF output")
	certificatePath := flag.String("certificate", "", "signing certificate in PEM format (required)")
	privateKeyPath := flag.String("private-key", "", "private key in PEM format (required)")
	chainPath := flag.String("chain", "", "optional PEM file containing intermediate/root certificates")
	flag.Parse()
	if *certificatePath == "" {
		fail("missing certificate", errors.New("use -certificate to provide the signing certificate"))
	}
	if *privateKeyPath == "" {
		fail("missing private key", errors.New("use -private-key to provide the signing key"))
	}

	certificate, err := loadCertificate(*certificatePath)
	if err != nil {
		fail("load certificate", err)
	}
	signer, err := loadPrivateKey(*privateKeyPath)
	if err != nil {
		fail("load private key", err)
	}
	chain, err := loadCertificates(*chainPath)
	if err != nil {
		fail("load certificate chain", err)
	}

	document, err := pdfsign.OpenFile(*input)
	if err != nil {
		fail("open input PDF", err)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fail("create output directory", err)
	}
	outputFile, err := os.Create(*output)
	if err != nil {
		fail("create output PDF", err)
	}
	defer outputFile.Close()

	document.Sign(signer, certificate, chain...).
		Type(pdfsign.CertificationSignature).
		SignerName(certificate.Subject.CommonName).
		Reason("Certificate signing test").
		Location("SMKVision")
	result, err := document.Write(outputFile)
	if err != nil {
		fail("sign PDF", err)
	}

	if len(result.Signatures) != 1 {
		fail("check signing result", fmt.Errorf("expected 1 signature, got %d", len(result.Signatures)))
	}

	verifyDocument, err := pdfsign.OpenFile(*output)
	if err != nil {
		fail("open signed PDF", err)
	}
	verification := verifyDocument.Verify().TrustSelfSigned(true)
	if !verification.Valid() {
		for index, signature := range verification.Signatures() {
			for _, verificationError := range signature.Errors {
				fmt.Fprintf(os.Stderr, "signature %d: %v\n", index+1, verificationError)
			}
		}
		fail("verify signed PDF", verification.Err())
	}

	fmt.Printf("signed and verified %s as %s\n", *output, certificate.Subject.CommonName)
}

func loadCertificate(path string) (*x509.Certificate, error) {
	certificates, err := loadCertificates(path)
	if err != nil {
		return nil, err
	}
	if len(certificates) == 0 {
		return nil, errors.New("no certificate found")
	}
	return certificates[0], nil
}

func loadCertificates(path string) ([]*x509.Certificate, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var certificates []*x509.Certificate
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		data = rest
		if block.Type != "CERTIFICATE" {
			continue
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, certificate)
	}
	return certificates, nil
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
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, errors.New("unsupported private key format")
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}
