package main

import (
	"bytes"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestTamperPDF(t *testing.T) {
	dir, root := testutil.Fixture(t, 4)
	input := filepath.Join(dir, "signed.pdf")
	output := filepath.Join(dir, "changed.pdf")
	original, _ := os.ReadFile(input)
	var out bytes.Buffer
	if err := tamperPDF(input, output, root, &out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(input)
	if !bytes.Equal(data, original) {
		t.Fatal("original changed")
	}
	if !strings.Contains(out.String(), "900 USD") {
		t.Fatal(out.String())
	}
	pool, _ := demo.Roots(root)
	if demo.Check(output, pool).Valid {
		t.Fatal("tampered output accepted")
	}
	if err := tamperPDF(filepath.Join(dir, "tampered.pdf"), output, root, &out); err == nil {
		t.Fatal("bad original accepted")
	}
	if err := tamperPDF(input, dir, root, &out); err == nil {
		t.Fatal("output directory accepted")
	}
	if err := tamperPDF(input, output, "missing.pem", &out); err == nil {
		t.Fatal("missing root accepted")
	}
}
