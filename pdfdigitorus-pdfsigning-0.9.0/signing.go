package pdfsigning

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/digitorus/pdfsign/sign"
	"github.com/digitorus/pdfsign/verify"
	gofpdf "github.com/jung-kurt/gofpdf"
)

// GenerateMockPDF writes a deterministic one-page document for signing tests.
func GenerateMockPDF(outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("pdfsign v0.9.0 mock document", true)
	pdf.SetAuthor("pdfdigitorus-pdfsigning-0.9.0", true)
	pdf.SetCreator("pdfdigitorus-pdfsigning-0.9.0", true)
	pdf.SetSubject("Mock PDF for digitorus/pdfsign v0.9.0", true)
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(0, 12, "PDF Signing Test Document", "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 7, "This one-page document is generated locally to test PDF signing with digitorus/pdfsign v0.9.0.", "", "L", false)
	pdf.Ln(8)
	pdf.SetDrawColor(80, 80, 80)
	pdf.SetFillColor(245, 247, 250)
	pdf.Rect(20, 75, 170, 42, "DF")
	pdf.SetFont("Arial", "B", 12)
	pdf.SetXY(28, 84)
	pdf.Cell(45, 8, "Signing API:")
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(110, 8, "sign.SignFile")
	pdf.SetXY(28, 96)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(45, 8, "Verification API:")
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(110, 8, "verify.VerifyWithOptions")

	if err := pdf.OutputFileAndClose(outputPath); err != nil {
		return fmt.Errorf("write mock PDF: %w", err)
	}
	return nil
}

// SignPDF applies a v0.9.0 certification signature without a certificate chain.
func SignPDF(inputPath, outputPath string, certificate *x509.Certificate, signer crypto.Signer) error {
	return SignPDFWithChain(inputPath, outputPath, certificate, signer, nil)
}

// SignPDFWithChain applies a v0.9.0 certification signature and embeds chain certificates.
func SignPDFWithChain(inputPath, outputPath string, certificate *x509.Certificate, signer crypto.Signer, chain []*x509.Certificate) error {
	if certificate == nil {
		return fmt.Errorf("certificate is required")
	}
	if signer == nil {
		return fmt.Errorf("signer is required")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	certificateChains := [][]*x509.Certificate(nil)
	if len(chain) > 0 {
		certificateChains = [][]*x509.Certificate{append([]*x509.Certificate{certificate}, chain...)}
	}

	err := sign.SignFile(inputPath, outputPath, sign.SignData{
		Signature: sign.SignDataSignature{
			CertType:   sign.CertificationSignature,
			DocMDPPerm: sign.AllowFillingExistingFormFieldsAndSignaturesPerms,
			Info: sign.SignDataSignatureInfo{
				Name:     certificate.Subject.CommonName,
				Location: "local test",
				Reason:   "v0.9.0 PDF signing test",
				Date:     time.Now().Local(),
			},
		},
		Signer:            signer,
		DigestAlgorithm:   crypto.SHA256,
		Certificate:       certificate,
		CertificateChains: certificateChains,
	})
	if err != nil {
		return fmt.Errorf("sign PDF: %w", err)
	}
	return nil
}

// VerifyPDF verifies a v0.9.0 signed PDF. AllowUntrustedRoots is intentional
// for this local self-signed test sample; production code should use a trust
// store instead.
func VerifyPDF(path string) (*verify.Response, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open signed PDF: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat signed PDF: %w", err)
	}
	options := verify.DefaultVerifyOptions()
	options.AllowUntrustedRoots = true
	response, err := verify.VerifyWithOptions(file, info.Size(), options)
	if err != nil {
		return nil, fmt.Errorf("verify PDF: %w", err)
	}
	return response, nil
}
