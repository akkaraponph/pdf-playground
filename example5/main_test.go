package main

import (
	"bytes"
	"path/filepath"
	"pdf-playground/internal/testutil"
	"strings"
	"testing"
)

func TestVerifyWithRoot(t *testing.T) {
	dir, root := testutil.Fixture(t, 5)
	_, wrongRoot := testutil.Fixture(t, 5)
	for _, tc := range []struct {
		name, root string
		ok         bool
	}{{"signed.pdf", root, true}, {"tampered.pdf", root, false}, {"unsigned.pdf", root, false}, {"missing.pdf", root, false}, {"signed.pdf", wrongRoot, false}, {"signed.pdf", "missing.pem", false}} {
		t.Run(tc.name+tc.root, func(t *testing.T) {
			var out bytes.Buffer
			err := verifyWithRoot(filepath.Join(dir, tc.name), tc.root, &out)
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v", err)
			}
			if tc.ok && !strings.Contains(out.String(), "passed strict") {
				t.Fatal(out.String())
			}
		})
	}
}
