package demo

import (
	"crypto/x509/pkix"
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageIdentityRoundTripAndReuse(t *testing.T) {
	for _, subject := range []pkix.Name{{CommonName: "สมชาย ใจดี", Organization: []string{"บริษัท ทดสอบลายมือชื่อ จำกัด"}, Country: []string{"TH"}}, {CommonName: "张伟", Organization: []string{"测试签名有限公司"}, Country: []string{"CN"}}} {
		t.Run(subject.CommonName, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "certs")
			id, err := LanguageIdentity(dir, subject)
			if err != nil {
				t.Fatal(err)
			}
			if id.Certificate.Subject.CommonName != subject.CommonName || id.Certificate.Subject.Organization[0] != subject.Organization[0] {
				t.Fatal(id.Certificate.Subject)
			}
			key, err := os.ReadFile(filepath.Join(dir, "signer.key.pem"))
			if err != nil {
				t.Fatal(err)
			}
			again, err := LanguageIdentity(dir, subject)
			if err != nil {
				t.Fatal(err)
			}
			if !again.Certificate.Equal(id.Certificate) {
				t.Fatal("existing certificate replaced")
			}
			after, _ := os.ReadFile(filepath.Join(dir, "signer.key.pem"))
			if string(after) != string(key) {
				t.Fatal("key replaced")
			}
			info, _ := os.Stat(filepath.Join(dir, "signer.key.pem"))
			if info.Mode().Perm() != 0600 {
				t.Fatal("key permissions")
			}
			if _, err := LanguageIdentity(dir, pkix.Name{CommonName: "wrong subject"}); err == nil {
				t.Fatal("wrong subject accepted")
			}
		})
	}
}
func TestLanguageIdentityRejectsPartialAndCorruptCredentials(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "signer.key.pem"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LanguageIdentity(dir, pkix.Name{CommonName: "张伟"}); err == nil {
		t.Fatal("partial credentials replaced")
	}
	dir = t.TempDir()
	subject := pkix.Name{CommonName: "สมชาย ใจดี"}
	if _, err := LanguageIdentity(dir, subject); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "signer.key.pem"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LanguageIdentity(dir, subject); err == nil {
		t.Fatal("corrupt key accepted")
	}
}
