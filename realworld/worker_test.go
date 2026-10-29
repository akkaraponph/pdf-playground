package realworld_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"pdf-playground/realworld/internal/adapter/filestore"
	"pdf-playground/realworld/internal/adapter/pdfgenerator"
	"pdf-playground/realworld/internal/adapter/pdfsigner"
	"pdf-playground/realworld/internal/document"
)

type failGeneratorOnce struct {
	document.Generator
	failed atomic.Bool
}

func (g *failGeneratorOnce) Generate(ctx context.Context, c document.Contract) ([]byte, error) {
	if !g.failed.Swap(true) {
		return nil, errors.New("injected render failure")
	}
	return g.Generator.Generate(ctx, c)
}

type failSignerOnce struct {
	document.Signer
	failed atomic.Bool
}

func (s *failSignerOnce) Sign(ctx context.Context, b []byte, reason string) ([]byte, error) {
	if !s.failed.Swap(true) {
		return nil, errors.New("injected signing failure")
	}
	return s.Signer.Sign(ctx, b, reason)
}

// This catches workers that lose failures, expose partial output, or strand
// tasks persisted in a running state after a process crash.
func TestBackgroundRecoveryAndRetry(t *testing.T) {
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
	s := document.NewService(store, &failGeneratorOnce{Generator: pdfgenerator.New()}, &failSignerOnce{Signer: signer})
	d, err := s.Generate(context.Background(), document.Contract{Title: "Contract", Customer: "Acme", AmountCents: 100, Currency: "USD"}, "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	// Model a crash after claim. Run must requeue this before executing it.
	d.Status = document.Generating
	if err := store.Save(context.Background(), d, d.Revision); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	failed := waitTask(t, s, d.ID, document.GenerationFailed)
	if len(failed.OriginalPDF) != 0 || failed.LastError == "" {
		t.Fatal("generation failure exposed partial output or no error")
	}
	found := false
	for _, event := range failed.Events {
		if event.Action == "recovered" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing recovery event")
	}
	if _, err := s.Retry(context.Background(), d.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	draft := waitTask(t, s, d.ID, document.Draft)
	if draft.LastError != "" || len(draft.OriginalPDF) == 0 {
		t.Fatal("retry did not complete generation")
	}
	if _, err := s.Submit(context.Background(), d.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Approve(context.Background(), d.ID, "Approved", "reviewer"); err != nil {
		t.Fatal(err)
	}
	failed = waitTask(t, s, d.ID, document.SigningFailed)
	if len(failed.SignedPDF) != 0 || failed.OriginalSHA256 != draft.OriginalSHA256 {
		t.Fatal("failed signing changed original or exposed signed output")
	}
	if _, err := s.Retry(context.Background(), d.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}
	waitTask(t, s, d.ID, document.Signed)
}
