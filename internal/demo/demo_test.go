package demo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerificationScenarios(t *testing.T) {
	dir := t.TempDir()
	if err := Generate(dir); err != nil {
		t.Fatal(err)
	}
	roots, err := Roots(filepath.Join(dir, "root.pem"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, file     string
		trusted, valid bool
	}{
		{"trusted", "signed.pdf", true, true},
		{"untrusted", "signed.pdf", false, false},
		{"tampered", "tampered.pdf", true, false},
		{"unsigned", "unsigned.pdf", true, false},
		{"missing", "missing.pdf", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := roots
			if !tc.trusted {
				pool = nil
			}
			r := Check(filepath.Join(dir, tc.file), pool)
			if tc.name == "tampered" {
				if len(r.Signatures) != 1 || !strings.Contains(strings.Join(r.Signatures[0].Errors, " "), "Message digest mismatch") {
					t.Fatalf("expected integrity failure: %+v", r)
				}
			}
			if r.Valid != tc.valid {
				t.Fatalf("valid=%v want %v: %+v", r.Valid, tc.valid, r)
			}
			if !tc.valid && len(r.Errors) == 0 && len(r.Signatures) == 0 {
				t.Fatal("missing diagnostics")
			}
		})
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.pem"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Roots(filepath.Join(dir, "bad.pem")); err == nil {
		t.Fatal("invalid root accepted")
	}
}

func TestTimestampFailureDoesNotWriteOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "TSA unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	dir := t.TempDir()
	input := filepath.Join(dir, "unsigned.pdf")
	output := filepath.Join(dir, "signed.pdf")
	if err := os.WriteFile(input, PDF(), 0644); err != nil {
		t.Fatal(err)
	}
	id, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := Sign(input, output, id, server.URL); err == nil {
		t.Fatal("TSA failure silently accepted")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("failed signing created output: %v", err)
	}
}

func TestExampleLabels(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateExample(dir, filepath.Join(dir, "root.pem"), 7); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ file, label string }{
		{"unsigned.pdf", "Unsigned document"},
		{"signed.pdf", "Case state: ORIGINAL"},
		{"tampered.pdf", "Signed document - modified content"},
	} {
		data, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Fatal(err)
		}
		for _, label := range []string{"Example 7", "JSON verification report", tc.label} {
			if !strings.Contains(string(data), label) {
				t.Fatalf("%s missing %s", tc.file, label)
			}
		}
	}
	roots, err := Roots(filepath.Join(dir, "root.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if !Check(filepath.Join(dir, "signed.pdf"), roots).Valid {
		t.Fatal("labels broke valid signature")
	}
	if Check(filepath.Join(dir, "tampered.pdf"), roots).Valid {
		t.Fatal("tampered label accepted")
	}
}

func TestGenerateSeparatesCertificates(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "example5")
	root := filepath.Join(dir, "certs", "root.pem")
	if err := GenerateExample(dir, root, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "root.pem")); !os.IsNotExist(err) {
		t.Fatal("certificate written into example folder")
	}
	pool, err := Roots(root)
	if err != nil {
		t.Fatal(err)
	}
	if !Check(filepath.Join(dir, "signed.pdf"), pool).Valid {
		t.Fatal("centralized root does not verify fixture")
	}
}
