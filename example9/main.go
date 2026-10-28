package main

import (
	"bytes"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/digitorus/pdfsign"
	"pdf-playground/internal/demo"
)

func main() {
	if err := run(".", os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(dir string, out io.Writer) error {
	subject := pkix.Name{CommonName: "สมชาย ใจดี", Organization: []string{"บริษัท ทดสอบลายมือชื่อ จำกัด"}, Country: []string{"TH"}}
	id, err := demo.LanguageIdentity(filepath.Join(dir, "certs"), subject)
	if err != nil {
		return err
	}
	input := filepath.Join(dir, "unsigned.pdf")
	output := filepath.Join(dir, "signed.pdf")
	source := demo.PDFFor(9, "Signed document - original content", "ACCEPTED with the matching test CA")
	if err := os.WriteFile(input, source, 0644); err != nil {
		return err
	}
	doc, err := pdfsign.OpenFile(input)
	if err != nil {
		return err
	}
	// SignerName and Reason also test Unicode PDF signature metadata, separately from X.509.
	doc.Sign(id.Key, id.Certificate, id.Root).
		Format(pdfsign.PAdES_B_B).
		SignerName(subject.CommonName).
		Reason("ทดสอบการลงนามเอกสารภาษาไทย")
	var signed bytes.Buffer
	if _, err := doc.Write(&signed); err != nil {
		return err
	}
	if err := os.WriteFile(output, signed.Bytes(), 0644); err != nil {
		return err
	}
	changed, err := demo.Tamper(signed.Bytes())
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "tampered.pdf"), changed, 0644); err != nil {
		return err
	}
	// Use a separate unsigned case label after creating the signed output.
	if err := os.WriteFile(input, demo.PDFFor(9, "Unsigned document", "REJECTED - no signatures"), 0644); err != nil {
		return err
	}
	roots, err := demo.Roots(filepath.Join(dir, "certs", "root.pem"))
	if err != nil {
		return err
	}
	result := demo.Check(output, roots)
	if !result.Valid || len(result.Signatures) != 1 || result.Signatures[0].Signer != subject.CommonName {
		return fmt.Errorf("Thai signer verification failed: %+v", result)
	}
	if _, err := fmt.Fprintf(out, "Certificate subject: %s\nSignature signer: %s\nSignature reason: ทดสอบการลงนามเอกสารภาษาไทย\nPDF passed strict verification with certs/root.pem\nSigned: signed.pdf; modified rejection case: tampered.pdf\n", id.Certificate.Subject.String(), result.Signatures[0].Signer); err != nil {
		return err
	}
	return nil
}
