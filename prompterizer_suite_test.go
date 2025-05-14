package prompterizer_test

import (
	"io"
	"log"
	"os"

	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/format"
)

func TestIntegration(t *testing.T) {
	stdOutErrToGinkgoWriter()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Prompterizer Test Suite")
}

func stdOutErrToGinkgoWriter() {
	MaxLength = 100000
	UseStringerRepresentation = true

	// Use os.Pipe to set up a set of files. Anything written to w can be read from r
	r, w, _ := os.Pipe()
	// Redorect os.Stdout to the pipe writer
	os.Stdout = w
	os.Stderr = w
	log.SetOutput(w)

	// Kick off a goroutine in the background to read from the pipe reader, and copy that content to the Ginkgowriter
	go func() {
		if _, err := io.Copy(GinkgoWriter, r); err != nil {
			log.Printf("error copying to GinkgoWriter: %v", err)
		}
		r.Close()
	}()
}
