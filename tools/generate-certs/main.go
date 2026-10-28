// Generate local test signing credentials. These are never production credentials.
package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"pdf-playground/internal/demo"
)

func main() {
	output := flag.String("output", "certs", "directory for local test credentials")
	flag.Parse()
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	names := []string{"signer.key.pem", "signer.crt.pem", "root.crt.pem", "chain.pem"}
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return fmt.Errorf("refusing to overwrite %s", name)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	id, err := demo.NewIdentity()
	if err != nil {
		return err
	}
	key, err := x509.MarshalPKCS8PrivateKey(id.Key)
	if err != nil {
		return err
	}
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: id.Certificate.Raw})
	root := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: id.Root.Raw})
	data := [][]byte{pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), cert, root, append(append([]byte{}, cert...), root...)}
	for i, name := range names {
		// Exclusive creation prevents accidental credential replacement.
		f, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := f.Write(data[i])
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	fmt.Printf("Created local test CA and signer in %s (RSA 2048, PKCS#8 key)\n", dir)
	return nil
}
