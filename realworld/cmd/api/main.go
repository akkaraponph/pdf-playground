package main

import (
	"context"
	"crypto/x509/pkix"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"pdf-playground/internal/demo"
	"pdf-playground/realworld/internal/adapter/filestore"
	"pdf-playground/realworld/internal/adapter/httpapi"
	"pdf-playground/realworld/internal/adapter/pdfgenerator"
	"pdf-playground/realworld/internal/adapter/pdfsigner"
	"pdf-playground/realworld/internal/document"
)

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	projectDir, err := findRealworldDir()
	if err != nil {
		return err
	}
	certDir := filepath.Join(projectDir, "certs", "api-demo")
	addr := flag.String("addr", "127.0.0.1:3000", "HTTP listen address")
	data := flag.String("data", filepath.Join(projectDir, "data"), "exclusive local data directory")
	cert := flag.String("certificate", filepath.Join(certDir, "signer.crt.pem"), "PEM leaf certificate followed by optional intermediates")
	key := flag.String("private-key", filepath.Join(certDir, "signer.key.pem"), "unencrypted PKCS#8 PEM key")
	root := flag.String("trusted-roots", filepath.Join(certDir, "root.pem"), "PEM root bundle")
	actor := flag.String("actor", "reviewer@example.com", "identity associated with local workflow actions")
	flag.Parse()
	if *actor == "" {
		return fmt.Errorf("-actor must not be empty")
	}
	if _, err := os.Stat(*cert); errors.Is(err, os.ErrNotExist) {
		if filepath.Clean(*cert) != filepath.Join(certDir, "signer.crt.pem") || filepath.Clean(*key) != filepath.Join(certDir, "signer.key.pem") || filepath.Clean(*root) != filepath.Join(certDir, "root.pem") {
			return fmt.Errorf("custom signing paths do not exist; provide all three credential files")
		}
		if _, err := demo.LanguageIdentity(certDir, pkix.Name{CommonName: "PDF signing local demo signer"}); err != nil {
			return fmt.Errorf("create local test identity: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check signing certificate: %w", err)
	}
	signer, err := pdfsigner.New(*cert, *key, *root)
	if err != nil {
		return fmt.Errorf("load signing identity: %w", err)
	}
	store, err := filestore.Open(*data)
	if err != nil {
		return err
	}
	defer store.Close()
	service := document.NewService(store, pdfgenerator.New(), signer)
	app := httpapi.New(service, *actor)
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	workerDone := make(chan error, 1)
	go func() { workerDone <- service.Run(workerCtx) }()
	serverDone := make(chan error, 1)
	go func() { serverDone <- app.Listen(*addr) }()
	slog.Info("PDF signing API starting", "address", *addr, "data", *data, "actor", *actor)
	var runErr error
	workerStopped, serverStopped := false, false
	select {
	case <-signals.Done():
	case err := <-workerDone:
		workerStopped = true
		runErr = fmt.Errorf("background worker stopped: %w", err)
	case err := <-serverDone:
		serverStopped = true
		runErr = err
	}
	// Drain HTTP first, then cancel and join the worker before releasing the store.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	shutdownErr := app.ShutdownWithContext(shutdownCtx)
	cancelWorker()
	if !workerStopped {
		runErr = errors.Join(runErr, <-workerDone)
	}
	if !serverStopped {
		runErr = errors.Join(runErr, <-serverDone)
	}
	return errors.Join(runErr, shutdownErr)
}

func findRealworldDir() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for _, candidate := range []string{workingDir, filepath.Join(workingDir, "realworld")} {
		if _, err := os.Stat(filepath.Join(candidate, "cmd", "api", "main.go")); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("run the API from the repository root or realworld directory")
}
