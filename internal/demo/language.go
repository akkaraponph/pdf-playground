package demo

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// LanguageIdentity creates or reuses a test identity in an ignored certs folder.
// Partial, malformed, mismatched, or differently named credentials are rejected.
func LanguageIdentity(dir string, subject pkix.Name) (*Identity, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	ignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(ignore); os.IsNotExist(err) {
		if err := os.WriteFile(ignore, []byte("*\n!.gitignore\n!README.md\n"), 0644); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	names := []string{"signer.key.pem", "signer.crt.pem", "root.pem"}
	count := 0
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			count++
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if count != 0 && count != len(names) {
		return nil, fmt.Errorf("partial credentials in %s; refusing to replace existing files", dir)
	}
	if count == 0 {
		id, err := NewIdentityForSubject(subject)
		if err != nil {
			return nil, err
		}
		key, err := x509.MarshalPKCS8PrivateKey(id.Key)
		if err != nil {
			return nil, err
		}
		blocks := []*pem.Block{{Type: "PRIVATE KEY", Bytes: key}, {Type: "CERTIFICATE", Bytes: id.Certificate.Raw}, {Type: "CERTIFICATE", Bytes: id.Root.Raw}}
		for i, name := range names {
			file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if err != nil {
				return nil, err
			}
			_, writeErr := file.Write(pem.EncodeToMemory(blocks[i]))
			closeErr := file.Close()
			if writeErr != nil {
				return nil, writeErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
		}
		return id, nil
	}
	blocks := make([]*pem.Block, len(names))
	for i, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid PEM in %s", name)
		}
		blocks[i] = block
	}
	if blocks[0].Type != "PRIVATE KEY" || blocks[1].Type != "CERTIFICATE" || blocks[2].Type != "CERTIFICATE" {
		return nil, fmt.Errorf("unexpected credential PEM types")
	}
	key, err := x509.ParsePKCS8PrivateKey(blocks[0].Bytes)
	if err != nil {
		return nil, err
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key is not a crypto.Signer")
	}
	cert, err := x509.ParseCertificate(blocks[1].Bytes)
	if err != nil {
		return nil, err
	}
	root, err := x509.ParseCertificate(blocks[2].Bytes)
	if err != nil {
		return nil, err
	}
	if cert.Subject.String() != subject.String() {
		return nil, fmt.Errorf("stored certificate subject %q does not match requested %q", cert.Subject.String(), subject.String())
	}
	if err := cert.CheckSignatureFrom(root); err != nil {
		return nil, fmt.Errorf("stored root does not issue signer: %w", err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(signer.Public())
	if err != nil {
		return nil, err
	}
	certKey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(publicKey, certKey) {
		return nil, fmt.Errorf("stored private key does not match certificate")
	}
	return &Identity{Key: signer, Certificate: cert, Root: root}, nil
}
