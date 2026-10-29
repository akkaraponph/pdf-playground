package document

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Service coordinates the document lifecycle through infrastructure ports.
type Service struct {
	repo      Repository
	generator Generator
	signer    Signer
	wake      chan struct{}
}

// NewService wires a document service without starting any goroutines.
func NewService(repo Repository, generator Generator, signer Signer) *Service {
	return &Service{repo: repo, generator: generator, signer: signer, wake: make(chan struct{}, 1)}
}

func (s *Service) notify() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func record(d *Document, status Status, action, actor, detail string) {
	d.Status = status
	d.UpdatedAt = time.Now().UTC()
	d.Events = append(d.Events, Event{action, actor, d.UpdatedAt, detail})
}
func newDocument(title, filename string) (Document, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Document{}, err
	}
	return Document{ID: hex.EncodeToString(id[:]), Title: title, Filename: filename, CreatedAt: time.Now().UTC()}, nil
}
func (s *Service) save(ctx context.Context, d Document) (Document, error) {
	if err := s.repo.Save(ctx, d, d.Revision); err != nil {
		return Document{}, err
	}
	d.Revision++
	return d, nil
}

// Generate queues a contract without doing PDF work on the request goroutine.
func (s *Service) Generate(ctx context.Context, c Contract, actor string) (Document, error) {
	if err := c.Validate(); err != nil {
		return Document{}, err
	}
	d, err := newDocument(c.Title, "contract.pdf")
	if err != nil {
		return d, err
	}
	d.Contract = &c
	record(&d, GenerationQueued, "generation_queued", actor, "")
	d, err = s.save(ctx, d)
	if err == nil {
		s.notify()
	}
	return d, err
}

// Upload accepts a parseable, unsigned PDF as a draft.
func (s *Service) Upload(ctx context.Context, title, filename string, data []byte, actor string) (Document, error) {
	if !validText(title, 150) || len(data) == 0 || len(data) > 10<<20 {
		return Document{}, fmt.Errorf("%w: title and PDF up to 10 MiB required", ErrInvalid)
	}
	if err := s.signer.Validate(data); err != nil {
		return Document{}, fmt.Errorf("%w: upload must be a parseable unsigned PDF", ErrInvalid)
	}
	filename = filepath.Base(strings.ReplaceAll(filename, "\\", "/"))
	if !validText(filename, 200) || !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename = "document.pdf"
	}
	d, err := newDocument(title, filename)
	if err != nil {
		return d, err
	}
	d.OriginalPDF = append([]byte(nil), data...)
	d.OriginalSHA256 = hash(data)
	record(&d, Draft, "uploaded", actor, "")
	return s.save(ctx, d)
}

// View retrieves metadata and stored PDF versions.
func (s *Service) View(ctx context.Context, id string) (Document, error) { return s.repo.Load(ctx, id) }

// List returns documents in creation order.
func (s *Service) List(ctx context.Context) ([]Document, error) { return s.repo.List(ctx) }

// Submit freezes a draft for approval.
func (s *Service) Submit(ctx context.Context, id, actor string) (Document, error) {
	d, err := s.View(ctx, id)
	if err != nil {
		return d, err
	}
	if d.Status != Draft {
		return d, ErrConflict
	}
	record(&d, Pending, "submitted", actor, "")
	return s.save(ctx, d)
}

// Approve queues server signing after an approval decision.
func (s *Service) Approve(ctx context.Context, id, reason, actor string) (Document, error) {
	if !validText(reason, 500) {
		return Document{}, fmt.Errorf("%w: approval reason required", ErrInvalid)
	}
	d, err := s.View(ctx, id)
	if err != nil {
		return d, err
	}
	if d.Status != Pending {
		return d, ErrConflict
	}
	d.Reason = reason
	d.ApprovedBy = actor
	record(&d, SigningQueued, "approved", actor, reason)
	d, err = s.save(ctx, d)
	if err == nil {
		s.notify()
	}
	return d, err
}

// Reject records a final negative decision.
func (s *Service) Reject(ctx context.Context, id, reason, actor string) (Document, error) {
	if !validText(reason, 500) {
		return Document{}, fmt.Errorf("%w: rejection reason required", ErrInvalid)
	}
	d, err := s.View(ctx, id)
	if err != nil {
		return d, err
	}
	if d.Status != Pending {
		return d, ErrConflict
	}
	d.Reason = reason
	record(&d, Rejected, "rejected", actor, reason)
	return s.save(ctx, d)
}

// Retry queues an explicitly failed task again without changing its input.
func (s *Service) Retry(ctx context.Context, id, actor string) (Document, error) {
	d, err := s.View(ctx, id)
	if err != nil {
		return d, err
	}
	var next Status
	switch d.Status {
	case GenerationFailed:
		next = GenerationQueued
	case SigningFailed:
		next = SigningQueued
	default:
		return d, ErrConflict
	}
	d.LastError = ""
	record(&d, next, "retried", actor, "")
	d, err = s.save(ctx, d)
	if err == nil {
		s.notify()
	}
	return d, err
}

// Verify checks the committed signed bytes with the configured trust roots.
func (s *Service) Verify(ctx context.Context, id string) (Verification, error) {
	d, err := s.View(ctx, id)
	if err != nil {
		return Verification{}, err
	}
	if d.Status != Signed {
		return Verification{}, ErrConflict
	}
	return s.signer.Verify(ctx, d.SignedPDF)
}
