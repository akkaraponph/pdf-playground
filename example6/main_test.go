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

func TestCompareTimestamps(t *testing.T) {
	dir, _ := testutil.Fixture(t, 6)
	id, _, _ := testutil.Credentials(t)
	tsa := testutil.TSA(t)
	timestamped := filepath.Join(dir, "timestamped.pdf")
	if err := demo.Sign(filepath.Join(dir, "unsigned.pdf"), timestamped, id, tsa.URL); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := compareTimestamps([]string{filepath.Join(dir, "signed.pdf"), timestamped}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "timestamp present=false") || !strings.Contains(out.String(), "timestamp present=true") {
		t.Fatal(out.String())
	}
	for _, file := range []string{"unsigned.pdf", "missing.pdf"} {
		if err := compareTimestamps([]string{filepath.Join(dir, file)}, &out); err == nil {
			t.Fatalf("%s accepted", file)
		}
	}
}
func TestLiveSigningAndTSAFailure(t *testing.T) {
	dir, _ := testutil.Fixture(t, 6)
	t.Chdir(dir)
	data, err := os.ReadFile("signed.pdf")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"baseline.pdf", "timestamped.pdf"} {
		if err := os.WriteFile(name, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	tsa := testutil.TSA(t)
	if err := run(tsa.URL); err != nil {
		t.Fatal(err)
	}
	report := demo.Check("timestamped-live.pdf", nil)
	if len(report.Signatures) != 1 || !report.Signatures[0].TimestampPresent {
		t.Fatal(report)
	}
	if err := run("http://127.0.0.1:1"); err == nil {
		t.Fatal("unreachable TSA accepted")
	}
}
