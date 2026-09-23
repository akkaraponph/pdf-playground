module pdf-signature-usecase

go 1.27.1

require github.com/digitorus/pdfsign v0.0.0-00010101000000-000000000000

require github.com/jung-kurt/gofpdf v1.16.3-0.20210918000319-0c885ad36193

require (
	github.com/digitorus/pdf v0.3.0 // indirect
	github.com/digitorus/pkcs7 v0.0.0-20260914070511-d678ea5ea03f // indirect
	github.com/digitorus/timestamp v0.0.0-20250524132541-c45532741eea // indirect
	github.com/mattetti/filebuffer v1.0.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/digitorus/pdfsign => /Users/akkaraponph/Workspaces/codespaces/akkaraponph/pdfsign

replace github.com/jung-kurt/gofpdf => github.com/refactorroom/gofpdf v1.16.3-0.20210918000319-0c885ad36193
