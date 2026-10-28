package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"pdf-playground/internal/testutil"
	"testing"
)

func TestVerifyBatch(t *testing.T) {
	dir, root := testutil.Fixture(t, 8)
	for _, tc := range []struct {
		names              []string
		accepted, rejected int
	}{{[]string{"signed.pdf"}, 1, 0}, {[]string{"signed.pdf", "tampered.pdf", "unsigned.pdf", "missing.pdf"}, 1, 3}} {
		var paths []string
		for _, name := range tc.names {
			paths = append(paths, filepath.Join(dir, name))
		}
		var out bytes.Buffer
		err := verifyBatch(paths, root, &out)
		var report batchSummary
		if e := json.Unmarshal(out.Bytes(), &report); e != nil {
			t.Fatal(e)
		}
		if report.Accepted != tc.accepted || report.Rejected != tc.rejected || (err != nil) != (tc.rejected > 0) {
			t.Fatalf("%+v err=%v", report, err)
		}
		if len(report.Documents) != len(paths) {
			t.Fatal("batch stopped before all inputs")
		}
	}
	if err := verifyBatch(nil, root, &bytes.Buffer{}); err == nil {
		t.Fatal("empty batch accepted")
	}
	if err := verifyBatch([]string{filepath.Join(dir, "signed.pdf")}, root, testutil.FailingWriter{}); err == nil {
		t.Fatal("write failure ignored")
	}
	if err := verifyBatch([]string{"missing.pdf"}, "missing.pem", &bytes.Buffer{}); err == nil {
		t.Fatal("missing roots ignored")
	}
}
