package prompterizer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/tenkeylabs/prompterizer"
)

type ResponseStruct struct {
	Value string `json:"value" prompt:"value,string" prompt_description:"The value to return"`
}

var _ = Describe("GenerateGeminiParts", func() {
	var (
		params prompterizer.PromptParams
	)

	BeforeEach(func() {
		params = prompterizer.PromptParams{
			SystemInstructions: []string{"System instruction 1", "System instruction 2"},
			Prompt:             []string{"Prompt 1", "Prompt 2"},
			FileCategory:       "Test Category",
			FileContent:        "Test content",
			FileData:           []byte("Test file data"),
			FileMimeType:       lo.ToPtr("text/plain"),
			ResponseStruct:     ResponseStruct{},
		}
	})

	It("should generate gemini parts when all fields are provided", func() {
		systemInstructions, parts, schema, err := prompterizer.GenerateGeminiParts(params)

		Expect(err).To(Not(HaveOccurred()))
		Expect(systemInstructions).To(Not(BeNil()))
		Expect(systemInstructions.Parts).To(HaveLen(2))
		Expect(systemInstructions.Parts[0].Text).To(Equal("System instruction 1"))
		Expect(systemInstructions.Parts[1].Text).To(Equal("System instruction 2"))

		Expect(parts).To(HaveLen(6))
		Expect(parts[0].Text).To(Equal("--- Test Category\n\n"))
		Expect(parts[1].Text).To(Equal("Test content"))
		Expect(parts[2].Text).To(Equal("\n\n---\n\n"))
		Expect(parts[3].InlineData.Data).To(Equal(params.FileData))
		Expect(parts[3].InlineData.MIMEType).To(Equal(*params.FileMimeType))
		Expect(parts[4].Text).To(Equal("Prompt 1"))
		Expect(parts[5].Text).To(Equal("Prompt 2"))

		Expect(schema).To(Not(BeNil()))
	})

	It("should return nil system instructions when none are provided", func() {
		params.SystemInstructions = nil

		systemInstructions, parts, schema, err := prompterizer.GenerateGeminiParts(params)

		Expect(err).To(Not(HaveOccurred()))

		Expect(systemInstructions).To(BeNil())
		Expect(parts).To(HaveLen(6))
		Expect(schema).To(Not(BeNil()))
	})

	It("should not include file content when FileCategory and FileContent are empty", func() {
		params.FileCategory = ""
		params.FileContent = ""

		systemInstructions, parts, schema, err := prompterizer.GenerateGeminiParts(params)

		Expect(err).To(Not(HaveOccurred()))
		Expect(systemInstructions).To(Not(BeNil()))
		Expect(systemInstructions.Parts).To(HaveLen(2))
		Expect(parts).To(HaveLen(3))
		Expect(parts[0].InlineData.Data).To(Equal(params.FileData))
		Expect(parts[0].InlineData.MIMEType).To(Equal(*params.FileMimeType))
		Expect(parts[1].Text).To(Equal("Prompt 1"))
		Expect(parts[2].Text).To(Equal("Prompt 2"))
		Expect(schema).To(Not(BeNil()))
	})

	It("should not include file data when FileData is nil", func() {
		params.FileData = nil
		params.FileMimeType = nil

		systemInstructions, parts, schema, err := prompterizer.GenerateGeminiParts(params)

		Expect(err).To(Not(HaveOccurred()))
		Expect(systemInstructions).To(Not(BeNil()))
		Expect(systemInstructions.Parts).To(HaveLen(2))
		Expect(parts).To(HaveLen(5))
		Expect(parts[0].Text).To(Equal("--- Test Category\n\n"))
		Expect(parts[1].Text).To(Equal("Test content"))
		Expect(parts[2].Text).To(Equal("\n\n---\n\n"))
		Expect(parts[3].Text).To(Equal("Prompt 1"))
		Expect(parts[4].Text).To(Equal("Prompt 2"))
		Expect(schema).To(Not(BeNil()))
	})

	It("should not include text prompts when none are provided", func() {
		params.Prompt = nil

		systemInstructions, parts, schema, err := prompterizer.GenerateGeminiParts(params)

		Expect(err).To(Not(HaveOccurred()))
		Expect(systemInstructions).To(Not(BeNil()))
		Expect(systemInstructions.Parts).To(HaveLen(2))
		Expect(parts).To(HaveLen(4))
		Expect(parts[0].Text).To(Equal("--- Test Category\n\n"))
		Expect(parts[1].Text).To(Equal("Test content"))
		Expect(parts[2].Text).To(Equal("\n\n---\n\n"))
		Expect(parts[3].InlineData.Data).To(Equal(params.FileData))
		Expect(parts[3].InlineData.MIMEType).To(Equal(*params.FileMimeType))
		Expect(schema).To(Not(BeNil()))
	})
})
