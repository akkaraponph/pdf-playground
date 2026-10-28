package demo

import (
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/digitorus/pdfsign"
)

// Report separates document errors and per-signature trust information.
type Report struct {
	File       string      `json:"file"`
	Valid      bool        `json:"valid"`
	Errors     []string    `json:"errors,omitempty"`
	Signatures []Signature `json:"signatures"`
}

// Signature is a compact presentation of the library's verification result.
type Signature struct {
	Signer           string   `json:"signer"`
	Valid            bool     `json:"valid"`
	TrustedChain     bool     `json:"trusted_chain"`
	TimestampPresent bool     `json:"timestamp_present"`
	TimestampValid   bool     `json:"timestamp_valid"`
	Errors           []string `json:"errors,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

// Roots loads public CA certificates into an explicit, separate trust pool.
func Roots(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no CA certificates in %s", path)
	}
	return pool, nil
}

// Check runs Strict with external fetching disabled for reproducible offline
// fixtures. This does not establish current revocation status in production.
func Check(path string, roots *x509.CertPool) Report {
	r := Report{File: path, Signatures: []Signature{}}
	doc, err := pdfsign.OpenFile(path)
	if err != nil {
		r.Errors = []string{err.Error()}
		return r
	}
	result := doc.Verify().Strict().TrustedRoots(roots).ExternalChecks(false).HTTPTimeout(5 * time.Second)
	if err := result.Err(); err != nil {
		r.Errors = append(r.Errors, err.Error())
	}
	for _, s := range result.Signatures() {
		r.Signatures = append(r.Signatures, Signature{Signer: s.SignerName, Valid: s.Valid, TrustedChain: s.TrustedChain, TimestampPresent: s.Timestamp != nil, TimestampValid: s.TimestampValid, Errors: messages(s.Errors), Warnings: messages(s.Warnings)})
	}
	if len(r.Signatures) == 0 {
		r.Errors = append(r.Errors, "no signatures found")
	}
	r.Valid = len(r.Errors) == 0 && len(r.Signatures) > 0 && result.Valid()
	return r
}

func messages(errors []error) []string {
	var out []string
	for _, err := range errors {
		out = append(out, err.Error())
	}
	return out
}
