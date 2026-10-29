// Package document owns the contract lifecycle and infrastructure ports.
package document

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Status describes the persisted document and background task state.
type Status string

const (
	GenerationQueued Status = "generation_queued"
	Generating       Status = "generating"
	GenerationFailed Status = "generation_failed"
	Draft            Status = "draft"
	Pending          Status = "pending_approval"
	SigningQueued    Status = "signing_queued"
	Signing          Status = "signing"
	SigningFailed    Status = "signing_failed"
	Signed           Status = "signed"
	Rejected         Status = "rejected"
)

var (
	ErrNotFound = errors.New("document not found")
	ErrConflict = errors.New("document state or revision conflict")
	ErrInvalid  = errors.New("invalid input")
)

// Contract holds the values rendered into a generated agreement.
type Contract struct {
	Title       string `json:"title"`
	Customer    string `json:"customer"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

// Validate restricts the sample template to its supported alphabet and currencies.
func (c Contract) Validate() error {
	if !validText(c.Title, 150) || !validText(c.Customer, 200) || c.AmountCents <= 0 || c.AmountCents > 1_000_000_000_00 {
		return fmt.Errorf("%w: title, customer and positive amount_cents are required (printable ASCII text)", ErrInvalid)
	}
	switch c.Currency {
	case "USD", "THB", "EUR":
	default:
		return fmt.Errorf("%w: currency must be USD, THB or EUR", ErrInvalid)
	}
	return nil
}

func validText(s string, max int) bool {
	if strings.TrimSpace(s) == "" || len(s) > max {
		return false
	}
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

// Event records a lifecycle action. It is not a tamper-proof ledger.
type Event struct {
	Action string    `json:"action"`
	Actor  string    `json:"actor"`
	At     time.Time `json:"at"`
	Detail string    `json:"detail,omitempty"`
}

// Document is the aggregate; PDF bytes are omitted from HTTP metadata responses.
type Document struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Filename       string    `json:"filename"`
	Status         Status    `json:"status"`
	Revision       int       `json:"revision"`
	Contract       *Contract `json:"contract,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	ApprovedBy     string    `json:"approved_by,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	OriginalSHA256 string    `json:"original_sha256,omitempty"`
	SignedSHA256   string    `json:"signed_sha256,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Events         []Event   `json:"events"`
	OriginalPDF    []byte    `json:"-"`
	SignedPDF      []byte    `json:"-"`
}

// Verification separates signature integrity/trust from online revocation checks.
type Verification struct {
	Valid             bool `json:"valid"`
	Signatures        int  `json:"signatures"`
	Trusted           bool `json:"trusted"`
	RevocationChecked bool `json:"revocation_checked"`
	TimestampChecked  bool `json:"timestamp_checked"`
}

// Repository saves complete aggregate snapshots with optimistic concurrency.
// expected=0 inserts; Save increments Revision only on successful commit.
type Repository interface {
	Load(context.Context, string) (Document, error)
	List(context.Context) ([]Document, error)
	Save(context.Context, Document, int) error
}

// Generator renders an example contract into PDF bytes.
type Generator interface {
	Generate(context.Context, Contract) ([]byte, error)
}

// Signer validates uploads, signs with the server identity and verifies PDFs.
type Signer interface {
	Validate([]byte) error
	Sign(context.Context, []byte, string) ([]byte, error)
	Verify(context.Context, []byte) (Verification, error)
}
