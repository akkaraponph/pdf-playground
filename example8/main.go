package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"pdf-playground/internal/demo"
)

type batchSummary struct {
	Accepted  int           `json:"accepted"`
	Rejected  int           `json:"rejected"`
	Documents []demo.Report `json:"documents"`
}

func main() {
	root := flag.String("roots", "certs/root.pem", "PEM trust roots")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"signed.pdf", "tampered.pdf", "unsigned.pdf"}
	}
	if err := verifyBatch(paths, *root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func verifyBatch(paths []string, root string, out io.Writer) error {
	if len(paths) == 0 {
		return fmt.Errorf("no PDF paths supplied")
	}
	roots, err := demo.Roots(root)
	if err != nil {
		return err
	}
	summary := batchSummary{Documents: make([]demo.Report, 0, len(paths))}
	for _, path := range paths {
		r := demo.Check(path, roots)
		summary.Documents = append(summary.Documents, r)
		if r.Valid {
			summary.Accepted++
		} else {
			summary.Rejected++
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		return err
	}
	if summary.Rejected > 0 {
		return fmt.Errorf("%d PDF(s) rejected; see JSON report", summary.Rejected)
	}
	return nil
}
