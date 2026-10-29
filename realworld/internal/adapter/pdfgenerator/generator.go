// Package pdfgenerator renders the example service agreement using gofpdf.
package pdfgenerator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/phpdave11/gofpdf"
	"pdf-playground/realworld/internal/document"
)

// Generator creates a fresh gofpdf instance for each task.
type Generator struct{}

var _ document.Generator = (*Generator)(nil)

// New creates the contract generator.
func New() *Generator { return &Generator{} }

// Generate renders a one-page service agreement suitable for digital signing.
func (*Generator) Generate(ctx context.Context, c document.Contract) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	// Keep the sample text inspectable for independent integrity/tamper tests.
	pdf.SetCompression(false)
	pdf.SetMargins(22, 24, 22)
	pdf.SetTitle(c.Title, false)
	pdf.SetAuthor("PDFSign showcase", false)
	pdf.AddPage()
	pdf.SetFillColor(24, 45, 70)
	pdf.Rect(0, 0, 210, 12, "F")
	pdf.SetTextColor(24, 45, 70)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.MultiCell(166, 11, c.Title, "", "L", false)
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(166, 9, "SERVICE AGREEMENT", "", 1, "L", false, 0, "")
	pdf.SetTextColor(45, 45, 45)
	pdf.SetFont("Helvetica", "", 12)
	for _, line := range []string{
		"Customer: " + c.Customer,
		fmt.Sprintf("Contract amount: %s %d.%02d", c.Currency, c.AmountCents/100, c.AmountCents%100),
		"",
		"The provider will deliver the agreed professional services to the customer. Payment is due within 30 days of delivery and acceptance.",
		"",
		"Approval workflow: review this document, submit it for approval, then approve to apply the server's digital signature or reject with a reason.",
		"",
		"The digital signature protects the PDF bytes. The application audit history separately records the reviewer's approval decision.",
	} {
		pdf.MultiCell(166, 7, line, "", "L", false)
		pdf.Ln(3)
	}
	pdf.SetY(258)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.MultiCell(166, 5, "Local showcase - generated with gofpdf and digitally signed with pdfsign. Test certificates require explicit trust. No handwritten signature is implied.", "", "L", false)
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
