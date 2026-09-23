package main

import (
	"flag"
	"fmt"
	"log"

	pdfsigning "pdfdigitorus-pdfsigning-0.9.0"
)

func main() {
	output := flag.String("output", "testdata/mock-document.pdf", "output PDF path")
	flag.Parse()

	if err := pdfsigning.GenerateMockPDF(*output); err != nil {
		log.Fatal(err)
	}
	fmt.Println(*output)
}
