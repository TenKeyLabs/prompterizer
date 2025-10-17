package prompterizer

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type PaginatedFileContent struct {
	PageNumber  int
	PageContent string
}

type PromptParams struct {
	SystemInstructions    []string
	Prompt                []string
	FileCategory          string
	FileContent           string                 // For text-based files (non-page-delineated)
	PaginatedFileContents []PaginatedFileContent // For paginated files (e.g., multi-page PDFs extracted to text)
	FileData              []byte
	FileMimeType          *string
	ResponseStruct        any
	TemplateVariables     map[string]string
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

	if params.FileCategory != "" && params.FileContent != "" {
		promptParts = append(promptParts, genai.NewPartFromText(fmt.Sprintf("--- %s\n\n", params.FileCategory)))
		promptParts = append(promptParts, genai.NewPartFromText(params.FileContent))
		promptParts = append(promptParts, genai.NewPartFromText("\n\n---\n\n"))
	}
	if len(params.PaginatedFileContents) > 0 && params.FileCategory != "" {
		promptParts = append(promptParts, generatePaginatedFileContentParts(params.PaginatedFileContents, params.FileCategory)...)
	}
	if params.FileData != nil && params.FileMimeType != nil {
		promptParts = append(promptParts, genai.NewPartFromBytes(params.FileData, *params.FileMimeType))
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
