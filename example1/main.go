package main

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/digitorus/pdfsign"
)

func main() {
	input := flag.String("input", "contract.pdf", "input PDF path")
	output := flag.String("output", "contract-signed.pdf", "signed PDF path")
	certPath := flag.String("certificate", "", "PEM certificate path")
	keyPath := flag.String("private-key", "", "unencrypted PEM private key path")
	tsaURL := flag.String("tsa", "", "RFC 3161 timestamp authority URL")
	flag.Parse()
	if *certPath == "" || *keyPath == "" || *tsaURL == "" || *input == *output {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "certificate, private-key and tsa are required; input and output must differ")
		os.Exit(2)
	}
	if err := run(*input, *output, *certPath, *keyPath, *tsaURL); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("signed PDF written to %s\n", *output)
}

func run(input, output, certPath, keyPath, tsaURL string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read certificate: %w", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return errors.New("expected a PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return errors.New("expected an unencrypted PEM private key")
	}
	var key any
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return fmt.Errorf("unsupported private key PEM type %q", block.Type)
	}
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return errors.New("private key does not implement crypto.Signer")
	}
	return signPDF(input, output, signer, cert, tsaURL)
}

func signPDF(input, output string, privateKey crypto.Signer, certificate *x509.Certificate, tsaURL string) error {
	inputInfo, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("stat input: %w", err)
	}
	outputInfo, err := os.Stat(output)
	if err == nil && os.SameFile(inputInfo, outputInfo) {
		return fmt.Errorf("input and output refer to the same file")
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat output: %w", err)
	}
	doc, err := pdfsign.OpenFile(input)
	if err != nil {
		return fmt.Errorf("open PDF: %w", err)
	}
	doc.Sign(privateKey, certificate).
		Format(pdfsign.PAdES_B_T).
		Timestamp(tsaURL).
		Reason("Contract approval")
	// Finish signing before touching the output, so TSA failures preserve it.
	var signed bytes.Buffer
	if _, err := doc.Write(&signed); err != nil {
		return fmt.Errorf("sign PDF: %w", err)
	}
	if err := os.WriteFile(output, signed.Bytes(), 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
