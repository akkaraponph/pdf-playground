// Package filestore implements a local, single-process document repository.
package filestore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"

	"pdf-playground/realworld/internal/document"
)

// Store serializes commits and holds an OS advisory lock for its lifetime.
// Supported on Unix (Linux/macOS); a shared network filesystem is not supported.
type Store struct {
	mu     sync.Mutex
	dir    string
	lock   *os.File
	closed bool
}

type snapshot struct {
	Document document.Document `json:"document"`
	Original []byte            `json:"original"`
	Signed   []byte            `json:"signed,omitempty"`
}

var _ document.Repository = (*Store)(nil)

// Open creates private storage and refuses a second owner of the directory.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("data directory already in use: %w", err)
	}
	return &Store{dir: dir, lock: f}, nil
}

// Directory returns the configured local data directory.
func (s *Store) Directory() string { return s.dir }

// Close releases exclusive ownership; callers must first stop workers and HTTP.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.lock.Close()
}

func validID(id string) bool {
	b, err := hex.DecodeString(id)
	return err == nil && len(b) == 16 && len(id) == 32
}
func (s *Store) load(id string) (document.Document, error) {
	if s.closed {
		return document.Document{}, os.ErrClosed
	}
	if !validID(id) {
		return document.Document{}, document.ErrNotFound
	}
	b, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return document.Document{}, document.ErrNotFound
	}
	if err != nil {
		return document.Document{}, err
	}
	var v snapshot
	if err := json.Unmarshal(b, &v); err != nil {
		return document.Document{}, fmt.Errorf("decode document %s: %w", id, err)
	}
	if v.Document.ID != id {
		return document.Document{}, fmt.Errorf("document ID mismatch")
	}
	v.Document.OriginalPDF = v.Original
	v.Document.SignedPDF = v.Signed
	return v.Document, nil
}

// Load returns a detached snapshot; callers cannot mutate stored bytes.
func (s *Store) Load(ctx context.Context, id string) (document.Document, error) {
	if err := ctx.Err(); err != nil {
		return document.Document{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load(id)
}

// List returns complete snapshots, oldest first. Intended for small demo datasets.
func (s *Store) List(ctx context.Context) ([]document.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, os.ErrClosed
	}
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	docs := make([]document.Document, 0, len(entries))
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		d, err := s.load(id)
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].CreatedAt.Before(docs[j].CreatedAt) })
	return docs, nil
}

// Save commits metadata, history and both PDF versions as one atomic snapshot.
func (s *Store) Save(ctx context.Context, d document.Document, expected int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return os.ErrClosed
	}
	if !validID(d.ID) {
		return document.ErrInvalid
	}
	existing, err := s.load(d.ID)
	if err != nil && !errors.Is(err, document.ErrNotFound) {
		return err
	}
	if (err == nil && existing.Revision != expected) || (err != nil && expected != 0) {
		return document.ErrConflict
	}
	if err == nil && expected == 0 {
		return document.ErrConflict
	}
	d.Revision = expected + 1
	data, err := json.Marshal(snapshot{d, d.OriginalPDF, d.SignedPDF})
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".snapshot-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), filepath.Join(s.dir, d.ID+".json")); err != nil {
		return err
	}
	// Persist the directory entry too; syncing the temporary file alone does
	// not make the rename survive a host crash on a local filesystem.
	dir, err := os.Open(s.dir)
	if err != nil {
		return fmt.Errorf("open directory after commit: %w", err)
	}
	syncErr := dir.Sync()
	closeErr := dir.Close()
	return errors.Join(syncErr, closeErr)
}
