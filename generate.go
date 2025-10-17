package prompterizer

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type FileContext struct {
	FileCategory string

	// pass one of
	BinaryFileContent     BinaryFileContent      // For Raw PDFs and other file types (multi-modal prompting)
	FileContent           string                 // For text-based files (non-page-delineated)
	PaginatedFileContents []PaginatedFileContent // For paginated files (e.g., multi-page PDFs extracted to text)
}

func (fc FileContext) IsEmpty() bool {
	return fc.FileContent == "" && len(fc.PaginatedFileContents) == 0 && len(fc.BinaryFileContent.FileBytes) == 0
}

type PaginatedFileContent struct {
	PageNumber  int
	PageContent string
}

type BinaryFileContent struct {
	FileBytes    []byte
	FileMimeType string
}

type PromptParams struct {
	SystemInstructions []string
	Prompt             []string
	FileContext        FileContext
	ResponseStruct     any
	TemplateVariables  map[string]string
}

type PromptSettings struct {
	Temperature float64
	TopP        float64
	TopK        int
	Candidates  int
}

type PromptGenerator[T any] interface {
	Generate(ctx context.Context) (T, error)
}

func GenerateGeminiParts(params PromptParams) (*genai.Content, []*genai.Part, *genai.Schema, error) {
	var systemInstruction *genai.Content
	for _, instruction := range params.SystemInstructions {
		if systemInstruction == nil {
			systemInstruction = &genai.Content{}
		}
		systemInstruction.Parts = append(systemInstruction.Parts, genai.NewPartFromText(instruction))
	}

	var promptParts []*genai.Part

	if !params.FileContext.IsEmpty() {
		var err error
		promptParts, err = generateFileContextParts(params.FileContext)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	for _, prompt := range params.Prompt {
		promptParts = append(promptParts, genai.NewPartFromText(prompt))
	}

	responseSchema, err := MarshalResponseSchema(params.ResponseStruct, params.TemplateVariables)
	if err != nil {
		return nil, nil, &genai.Schema{}, err
	}

	return systemInstruction, promptParts, responseSchema, nil
}

func generateFileContextParts(fileContext FileContext) ([]*genai.Part, error) {
	if fileContext.FileCategory == "" {
		return nil, fmt.Errorf("fileCategory is required")
	}

	var promptParts []*genai.Part
	switch {
	case fileContext.FileContent != "":
		promptParts = append(promptParts, generateFileContentParts(fileContext.FileContent, fileContext.FileCategory)...)
	case len(fileContext.PaginatedFileContents) > 0:
		promptParts = append(promptParts, generatePaginatedFileContentParts(fileContext.PaginatedFileContents, fileContext.FileCategory)...)
	case len(fileContext.BinaryFileContent.FileBytes) > 0:
		promptParts = append(promptParts, generateFileBytesParts(fileContext.BinaryFileContent, fileContext.FileCategory)...)
	default:
		return nil, fmt.Errorf("file content not specified")
	}

	return promptParts, nil
}

func generateFileContentParts(fileContent string, fileCategory string) []*genai.Part {
	var parts []*genai.Part
	if fileCategory != "" && fileContent != "" {
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("--- %s\n\n", fileCategory)))
		parts = append(parts, genai.NewPartFromText(fileContent))
		parts = append(parts, genai.NewPartFromText("\n\n---\n\n"))
	}
	return parts
}

func generatePaginatedFileContentParts(paginatedFileContents []PaginatedFileContent, fileCategory string) []*genai.Part {
	var parts []*genai.Part
	parts = append(parts, genai.NewPartFromText(fmt.Sprintf("--- %s\n\n", fileCategory)))
	for _, page := range paginatedFileContents {
		parts = append(parts, genai.NewPartFromText(fmt.Sprintf("------ (Page %d)\n\n", page.PageNumber)))
		parts = append(parts, genai.NewPartFromText(page.PageContent))
		parts = append(parts, genai.NewPartFromText("\n\n------\n\n"))
	}
	parts = append(parts, genai.NewPartFromText("\n\n---\n\n"))
	return parts
}

func generateFileBytesParts(binaryFile BinaryFileContent, fileCategory string) []*genai.Part {
	var parts []*genai.Part
	parts = append(parts, genai.NewPartFromText(fmt.Sprintf("--- %s\n\n", fileCategory)))
	parts = append(parts, genai.NewPartFromBytes(binaryFile.FileBytes, binaryFile.FileMimeType))
	parts = append(parts, genai.NewPartFromText("\n\n---\n\n"))
	return parts
}
