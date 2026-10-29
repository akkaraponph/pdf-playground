package document

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Run consumes the persisted backlog with one bounded worker. Call once in a
// background goroutine per process. The store must enforce exclusive ownership.
// A startup scan recovers tasks interrupted after claiming but before committing.
func (s *Service) Run(ctx context.Context) error {
	docs, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, d := range docs {
		next := d.Status
		switch d.Status {
		case Generating:
			next = GenerationQueued
		case Signing:
			next = SigningQueued
		default:
			continue
		}
		record(&d, next, "recovered", "worker", "interrupted task requeued on startup")
		if _, err := s.save(ctx, d); err != nil {
			return err
		}
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := s.drain(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// A failed commit remains running; stop rather than silently strand work.
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.wake:
		case <-ticker.C:
		}
	}
}

func (s *Service) drain(ctx context.Context) error {
	docs, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.Status != GenerationQueued && d.Status != SigningQueued {
			continue
		}
		if err := s.process(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) process(ctx context.Context, d Document) error {
	generating := d.Status == GenerationQueued
	running, success, failure := Signing, Signed, SigningFailed
	if generating {
		running, success, failure = Generating, Draft, GenerationFailed
	}
	record(&d, running, string(running), "worker", "")
	d, err := s.save(ctx, d)
	if err != nil {
		return err
	}
	taskCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var output []byte
	// Isolate malformed PDF panics to the task so the worker can record a failure.
	output, err = func() (out []byte, err error) {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("PDF task panic: %v", v)
			}
		}()
		if generating {
			if d.Contract == nil {
				return nil, fmt.Errorf("missing contract input")
			}
			return s.generator.Generate(taskCtx, *d.Contract)
		}
		return s.signer.Sign(taskCtx, d.OriginalPDF, d.Reason)
	}()
	if err == nil {
		err = taskCtx.Err()
	}
	if err != nil {
		slog.Error("document task failed", "document_id", d.ID, "stage", running, "error", err)
		d.LastError = "PDF task failed; inspect server logs and retry"
		record(&d, failure, string(failure), "worker", d.LastError)
	} else {
		if generating {
			d.OriginalPDF = output
			d.OriginalSHA256 = hash(output)
		} else {
			d.SignedPDF = output
			d.SignedSHA256 = hash(output)
		}
		d.LastError = ""
		record(&d, success, string(success), "worker", "")
	}
	// Finish the atomic commit even during shutdown; the main goroutine waits.
	commitCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer stop()
	_, saveErr := s.save(commitCtx, d)
	return saveErr
}
