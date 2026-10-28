package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"pdf-playground/internal/demo"
	"pdf-playground/internal/testutil"
	"testing"
)

func TestWriteReport(t *testing.T) {
	dir, root := testutil.Fixture(t, 7)
	for _, name := range []string{"signed.pdf", "tampered.pdf", "unsigned.pdf", "missing.pdf"} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			err := writeReport(filepath.Join(dir, name), root, &out)
			var r demo.Report
			if e := json.Unmarshal(out.Bytes(), &r); e != nil {
				t.Fatal(e)
			}
			want := name == "signed.pdf"
			if r.Valid != want || (err == nil) != want {
				t.Fatalf("%+v err=%v", r, err)
			}
		})
	}
	if err := writeReport(filepath.Join(dir, "signed.pdf"), root, testutil.FailingWriter{}); err == nil {
		t.Fatal("write failure ignored")
	}
	if err := writeReport(filepath.Join(dir, "signed.pdf"), "missing.pem", &bytes.Buffer{}); err == nil {
		t.Fatal("missing roots ignored")
	}
}
