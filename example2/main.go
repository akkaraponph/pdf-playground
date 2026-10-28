package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/digitorus/pdfsign"
)

func main() {
	input := flag.String("input", "contract-signed.pdf", "signed PDF path")
	flag.Parse()
	if err := verifyPDF(*input); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("PDF passed strict signature verification")
}

func verifyPDF(path string) error {
	doc, err := pdfsign.OpenFile(path)
	if err != nil {
		return fmt.Errorf("open PDF: %w", err)
	}

	result := doc.Verify().Strict()
	if err := result.Err(); err != nil {
		return fmt.Errorf("PDF signature verification failed: %w", err)
	}
	if result.Count() == 0 {
		return fmt.Errorf("PDF signature verification failed: no signatures found")
	}
	if !result.Valid() {
		return fmt.Errorf("PDF signature verification failed: %+v", result.Signatures())
	}
	return nil
}
