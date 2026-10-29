package realworld_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"pdf-playground/internal/demo"
	"pdf-playground/realworld/internal/adapter/filestore"
	"pdf-playground/realworld/internal/adapter/httpapi"
	"pdf-playground/realworld/internal/adapter/pdfgenerator"
	"pdf-playground/realworld/internal/adapter/pdfsigner"
	"pdf-playground/realworld/internal/document"
)

func credentials(t *testing.T) (string, string, string) {
	t.Helper()
	id, err := demo.NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	key, err := x509.MarshalPKCS8PrivateKey(id.Key)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"), filepath.Join(dir, "root.pem")}
	for i, b := range []*pem.Block{{Type: "CERTIFICATE", Bytes: id.Certificate.Raw}, {Type: "PRIVATE KEY", Bytes: key}, {Type: "CERTIFICATE", Bytes: id.Root.Raw}} {
		if err := os.WriteFile(paths[i], pem.EncodeToMemory(b), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return paths[0], paths[1], paths[2]
}

func request(t *testing.T, app *fiber.App, method, path, body string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func waitStatus(t *testing.T, s *document.Service, id string, want document.Status) document.Document {
	return waitTask(t, s, id, want)
}

func waitTask(t *testing.T, s *document.Service, id string, want document.Status) document.Document {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		d, err := s.View(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if d.Status == want {
			return d
		}
		if (d.Status == document.GenerationFailed || d.Status == document.SigningFailed) && want != document.GenerationFailed && want != document.SigningFailed {
			t.Fatalf("task failed: %+v", d)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", want)
	return document.Document{}
}

// This catches synchronous request work, broken lifecycle guards, lost persistence,
// missing lifecycle guards, and signatures whose bytes fail cryptographic verification.
func TestRealContractFlow(t *testing.T) {
	cert, key, root := credentials(t)
	signer, err := pdfsigner.New(cert, key, root)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	store, err := filestore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	s := document.NewService(store, pdfgenerator.New(), signer)
	app := httpapi.New(s, "reviewer@example.com")
	code, body := request(t, app, "POST", "/api/v1/documents/generate", `{"title":"Service agreement","customer":"Acme Ltd","amount_cents":125050,"currency":"USD"}`)
	if code != 202 {
		t.Fatalf("generate: %d %s", code, body)
	}
	var d document.Document
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatal(err)
	}
	if d.Status != document.GenerationQueued {
		t.Fatalf("not queued: %s", body)
	}
	base := "/api/v1/documents/" + d.ID
	code, _ = request(t, app, "POST", base+"/approve", `{"reason":"Approved"}`)
	if code != 409 {
		t.Fatalf("approve before submit=%d", code)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	joined := false
	defer func() {
		cancel()
		if !joined {
			if err := <-done; err != nil {
				t.Error(err)
			}
		}
	}()
	waitStatus(t, s, d.ID, document.Draft)
	code, pdf := request(t, app, "GET", base+"/download?version=original", "")
	if code != 200 || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("original: %d", code)
	}
	for _, step := range []struct {
		path, body string
		code       int
	}{{"submit", `{}`, 200}, {"approve", `{"reason":"Approved service agreement"}`, 202}, {"approve", `{"reason":"Approved again"}`, 409}} {
		code, b := request(t, app, "POST", base+"/"+step.path, step.body)
		if code != step.code {
			t.Fatalf("%s: %d %s", step.path, code, b)
		}
	}
	signedDoc := waitStatus(t, s, d.ID, document.Signed)
	if signedDoc.OriginalSHA256 == signedDoc.SignedSHA256 || len(signedDoc.Events) < 6 {
		t.Fatalf("missing hashes/history: %+v", signedDoc)
	}
	code, signed := request(t, app, "GET", base+"/download", "")
	if code != 200 {
		t.Fatalf("signed download=%d", code)
	}
	report, err := signer.Verify(context.Background(), signed)
	if err != nil || !report.Valid || report.Signatures != 1 {
		t.Fatalf("verification: %+v %v", report, err)
	}
	// Change visible text without shifting offsets; integrity must fail.
	tampered := bytes.Replace(signed, []byte("Acme Ltd"), []byte("Evil Ltd"), 1)
	if bytes.Equal(tampered, signed) {
		t.Fatal("customer text missing from generated PDF")
	}
	report, err = signer.Verify(context.Background(), tampered)
	if err == nil && report.Valid {
		t.Fatal("tampered contract accepted")
	}
	code, b := request(t, app, "GET", base+"/verify", "")
	if code != 200 || !bytes.Contains(b, []byte(`"valid":true`)) {
		t.Fatalf("verify: %d %s", code, b)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	joined = true
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := filestore.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restored, err := reopened.Load(context.Background(), d.ID)
	if err != nil || restored.Status != document.Signed || !bytes.Equal(restored.SignedPDF, signed) {
		t.Fatalf("restart lost signed document: %v", err)
	}
}

func TestValidationRejectionAndRevisionConflict(t *testing.T) {
	cert, key, root := credentials(t)
	signer, err := pdfsigner.New(cert, key, root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := filestore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if other, err := filestore.Open(store.Directory()); err == nil {
		other.Close()
		t.Fatal("second process owner accepted")
	}
	s := document.NewService(store, pdfgenerator.New(), signer)
	app := httpapi.New(s, "reviewer@example.com")
	for _, body := range []string{`{`, `{}`, `{"title":"x","customer":"A","amount_cents":-1,"currency":"USD"}`} {
		code, _ := request(t, app, "POST", "/api/v1/documents/generate", body)
		if code != 400 {
			t.Fatalf("invalid input=%d", code)
		}
	}
	_, err = s.Upload(context.Background(), "bad", "bad.pdf", []byte("%PDF-not really"), "reviewer@example.com")
	if err == nil {
		t.Fatal("malformed PDF accepted")
	}
	d, err := s.Upload(context.Background(), "Contract", "contract.pdf", demo.PDF(), "reviewer@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Submit(context.Background(), d.ID, "reviewer@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Reject(context.Background(), d.ID, "", "reviewer@example.com"); err == nil {
		t.Fatal("empty rejection accepted")
	}
	rejected, err := s.Reject(context.Background(), d.ID, "Incorrect amount", "reviewer@example.com")
	if err != nil || rejected.Status != document.Rejected {
		t.Fatalf("reject: %v", err)
	}
	if _, err = s.Approve(context.Background(), d.ID, "Approved", "reviewer@example.com"); err == nil {
		t.Fatal("rejected document approved")
	}
	snapshot, err := store.Load(context.Background(), d.ID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() { results <- store.Save(context.Background(), snapshot, snapshot.Revision) })
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent commits=%d, want 1", successes)
	}
	code, _ := request(t, app, "GET", "/api/v1/documents/"+fmt.Sprintf("%032d", 0), "")
	if code != 404 {
		t.Fatalf("missing doc=%d", code)
	}
}
