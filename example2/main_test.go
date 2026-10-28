package main

import (
	"os"
	"path/filepath"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestVerifyPDFRejectsUntrustedUnsignedAndChanged(t *testing.T) {
	dir, _ := testutil.Fixture(t, 2)
	for _, tc := range []struct{ name, contains string }{{"signed.pdf", "verification failed"}, {"unsigned.pdf", "verification failed"}, {"tampered.pdf", "verification failed"}, {"missing.pdf", "open PDF"}} {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyPDF(filepath.Join(dir, tc.name))
			if err == nil || !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("want %q got %v", tc.contains, err)
			}
		})
	}
	bad := filepath.Join(dir, "invalid.pdf")
	if err := os.WriteFile(bad, []byte("not a PDF"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyPDF(bad); err == nil {
		t.Fatal("invalid PDF accepted")
	}
}
