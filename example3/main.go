package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"pdf-playground/internal/demo"
)

func main() {
	input := flag.String("input", "contract-signed.pdf", "PDF to diagnose")
	flag.Parse()
	if err := diagnose(*input, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func diagnose(input string, out io.Writer) error {
	r := demo.Check(input, nil)
	fmt.Fprintf(out, "Document accepted: %v\n", r.Valid)
	for _, err := range r.Errors {
		fmt.Fprintln(out, "Document error:", err)
	}
	for i, s := range r.Signatures {
		fmt.Fprintf(out, "Signature %d: %s\n  valid=%v trusted-chain=%v timestamp-present=%v timestamp-valid=%v\n", i+1, s.Signer, s.Valid, s.TrustedChain, s.TimestampPresent, s.TimestampValid)
		for _, err := range s.Errors {
			fmt.Fprintln(out, "  Error:", err)
		}
		for _, warn := range s.Warnings {
			fmt.Fprintln(out, "  Warning:", warn)
		}
	}
	if len(r.Signatures) == 0 {
		return fmt.Errorf("no signatures available for diagnosis")
	}
	fmt.Fprintln(out, "Diagnosis complete. A rejected signature needs its certificate, usage, and timestamp errors investigated.")
	return nil
}
