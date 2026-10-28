package main

import (
	"fmt"
	"io"
	"os"
	"pdf-playground/internal/demo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	return tamperPDF("signed.pdf", "tampered.pdf", "certs/root.pem", os.Stdout)
}
func tamperPDF(input, output, root string, out io.Writer) error {
	roots, err := demo.Roots(root)
	if err != nil {
		return err
	}
	original := demo.Check(input, roots)
	if !original.Valid {
		return fmt.Errorf("original signature failed: %+v", original)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	data, err = demo.Tamper(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, data, 0644); err != nil {
		return err
	}
	modified := demo.Check(output, roots)
	if modified.Valid {
		return fmt.Errorf("changed signed content was unexpectedly accepted")
	}
	fmt.Fprintln(out, "Original signed PDF: accepted")
	fmt.Fprintln(out, "Changed amount from 100 USD to 900 USD: rejected")
	for _, s := range modified.Signatures {
		for _, e := range s.Errors {
			fmt.Fprintln(out, e)
		}
	}
	return nil
}
