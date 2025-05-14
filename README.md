# Prompterizer

Prompterizer is a Go library for simplifying structured prompt generation and response handling with Google's Gemini AI models. It converts Go structs with tags into `genai.Schema` for AI interaction.

## Features

- **Schema from Go Structs:** Define AI response formats using Go structs and tags.
- **Gemini-Ready:** Generates `*genai.Content` (system instructions), `[]*genai.Part` (prompts), and `*genai.Schema`.
- **Rich Struct Tags:** Customize field names, types, descriptions, requirements, and aliases.
- **Complex Structures:** Supports nested/embedded structs and slices.
- **Dynamic Descriptions:** Use template variables in field descriptions.
- **Easy Unmarshaling:** Helper to unmarshal JSON responses to Go structs.

## Installation

```bash
go get github.com/tenkeylabs/prompterizer
```

Dependencies (`google.golang.org/genai`, `github.com/samber/lo`) are handled by `go mod tidy`.

## Usage

### 1. Define Response Structure

Use Go structs and `prompt` tags:

```go
package main

type Address struct {
    Street string `prompt:"street,string,required" prompt_description:"Street name."`
    City   string `prompt:"city,string,required"`
}

type UserProfile struct {
    FullName    string  `prompt:"full_name,string,required" prompt_description:"User's full name."`
    Age         int     `prompt:"age,integer"`
    PrimaryAddr Address `prompt:"primary_address,object,required"`
}
```

### 2. Generate Prompt Components

```go
package main

import (
    "log"
    "github.com/tenkeylabs/prompterizer"
)

func main() {
    params := prompterizer.PromptParams{
        SystemInstructions: []string{"You are a helpful AI."},
        Prompt:             []string{"Extract user profile."},
        ResponseStruct:     prompterizer.UserProfile{}, // Or new(prompterizer.UserProfile)
        TemplateVariables:  map[string]string{"example_detail": "some value"},
    }

    _, _, responseSchema, err := prompterizer.GenerateGeminiParts(params)
    if err != nil {
        log.Fatalf("Error generating Gemini parts: %v", err)
    }
    // Use responseSchema with your Gemini model
}
```

### 3. Unmarshal AI Responses

```go
// jsonResponse is the JSON string from the AI
profile, err := prompterizer.Unmarshal[UserProfile](jsonResponse)
if err != nil {
    log.Fatalf("Failed to unmarshal: %v", err)
}
// Use profile
```

## Struct Tag Reference

- **`prompt:"<name>,<type>[,required]"`**:
  - `name`: JSON property name.
  - `type`: `string`, `bool`, `number`, `integer`, `array`, `object`.
  - `required`: (Optional) Marks field as required.
- **`prompt_description:"<text>"`**: (Optional) Field description. Supports `{var}` templating.
- **`prompt_aliases:"<alias1>,<alias2>"`**: (Optional) Alternative names, added to description.

## Contributing

Contributions are welcome! Please submit a PR or open an issue.

## License

Apache 2.0
