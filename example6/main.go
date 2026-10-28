package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
)

func main() {
	tsa := flag.String("tsa", "", "optional RFC 3161 endpoint to create a new B-T sample")
	flag.Parse()
	if err := run(*tsa); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(tsa string) error {
	paths := []string{"baseline.pdf", "timestamped.pdf"}
	if tsa != "" {
		id, err := demo.NewIdentity()
		if err != nil {
			return err
		}
		dir, err := os.MkdirTemp("", "example6-live-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)
		input := filepath.Join(dir, "source.pdf")
		if err := os.WriteFile(input, demo.PDFFor(6, "Live B-T signature with timestamp", "Timestamp present; check TSA trust separately"), 0600); err != nil {
			return err
		}
		if err := demo.Sign(input, "timestamped-live.pdf", id, tsa); err != nil {
			return fmt.Errorf("B-T signing failed (no B-B fallback): %w", err)
		}
		fmt.Println("Created timestamped-live.pdf using an ephemeral signer")
		paths = append(paths, "timestamped-live.pdf")
	}
	return compareTimestamps(paths, os.Stdout)
}
func compareTimestamps(paths []string, out io.Writer) error {
	for _, path := range paths {
		report := demo.Check(path, nil)
		if len(report.Errors) > 0 {
			return fmt.Errorf("read %s: %v", path, report.Errors)
		}
		if len(report.Signatures) != 1 {
			return fmt.Errorf("expected one signature in %s", path)
		}
		s := report.Signatures[0]
		fmt.Fprintf(out, "%s: timestamp present=%v; timestamp trusted=%v\n", path, s.TimestampPresent, s.TimestampValid)
	}
	fmt.Fprintln(out, "Timestamp presence alone does not establish TSA trust")
	return nil
}
