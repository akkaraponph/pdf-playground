package main

import (
	"bytes"
	"path/filepath"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestDiagnose(t *testing.T) {
	dir, _ := testutil.Fixture(t, 3)
	for _, tc := range []struct {
		name      string
		wantError bool
		contains  string
	}{
		{"signed.pdf", false, "trusted-chain=false"}, {"unsigned.pdf", true, "no signatures found"}, {"missing.pdf", true, "Document error:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := diagnose(filepath.Join(dir, tc.name), &out)
			if (err != nil) != tc.wantError || !strings.Contains(out.String(), tc.contains) {
				t.Fatalf("err=%v output=%s", err, &out)
			}
		})
	}
}
