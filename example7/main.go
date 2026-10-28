package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"pdf-playground/internal/demo"
)

func main() {
	input := flag.String("input", "signed.pdf", "PDF to report")
	root := flag.String("roots", "certs/root.pem", "PEM trust roots")
	flag.Parse()
	if err := writeReport(*input, *root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func writeReport(input, root string, out io.Writer) error {
	roots, err := demo.Roots(root)
	if err != nil {
		return err
	}
	report := demo.Check(input, roots)
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return err
	}
	if !report.Valid {
		return fmt.Errorf("PDF rejected; see JSON report")
	}
	return nil
}
