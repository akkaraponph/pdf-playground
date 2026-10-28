package main

import (
	"fmt"
	"github.com/digitorus/pdfsign"
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
	return verifyWithRoot("signed.pdf", "certs/root.pem", os.Stdout)
}
func verifyWithRoot(input, root string, out io.Writer) error {
	roots, err := demo.Roots(root)
	if err != nil {
		return err
	}
	doc, err := pdfsign.OpenFile(input)
	if err != nil {
		return err
	}
	// Configure all options before any result accessor triggers verification.
	result := doc.Verify().Strict().TrustedRoots(roots).ExternalChecks(false)
	if err := result.Err(); err != nil {
		return err
	}
	if result.Count() == 0 || !result.Valid() {
		return fmt.Errorf("local CA verification failed: %+v", result.Signatures())
	}
	fmt.Fprintln(out, "PDF passed strict certificate checks with the explicit playground root")
	fmt.Fprintln(out, "External revocation fetching is disabled for this offline fixture")
	return nil
}
