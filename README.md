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
    Country string `prompt:"country,string" prompt_enum:"US,CA,MX,INVALID"`
}

type UserProfile struct {
    FullName    string  `prompt:"full_name,string,required" prompt_description:"User's full name."`
    Age         *int     `prompt:"age,integer"` // Pointer values set the schema to Nullable
    DateOfBirth time.Time  `prompt:"dateOfBirth,string,date-time"`
    Addresses []Address `prompt:"addresses,object,required"`
    TemplatedDescription string `prompt:"templatedDescription,string" prompt_description:"This is a templated description with a variable: {example_detail}"`
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

- **`prompt:"<name>,<type>[,format][,required]"`**:
  - `name`: JSON property name.
  - `type`: `string`, `bool`, `number`, `integer`, `object`.
    - For array/slice fields, specify the expected type of the array items e.g.
    ```
    Addresses []Address `prompt:"addresses,object`
    Names []string `prompt:"names,string`
    ```
  - `format`: (Optional) Describe the expected format of the value to be returned as per [OpenApi 3.0 spec](https://spec.openapis.org/registry/format/#formats-registry) e.g. `date-time` (for ISO 8601)
    - If `prompt_enum` is present, format `enum` is automatically set if not explicitly overridden.
    - If type `number` is present, format `float` is automatically set if not explicitly overridden.
  - `required`: (Optional) Marks field as required.
- **`prompt_enum:"<value1>,<value2>,..."`**: (Optional) Specify an enumeration of possible return values in a comma-separated list.
  - Sets the format to `enum` if a format is not explicitly set.
- **`prompt_description:"<text>"`**: (Optional) Field description. Supports `{var}` templating.
- **`prompt_aliases:"<alias1>,<alias2>"`**: (Optional) Alternative names, added to description.
- If field is a pointer, it's marked as Nullable

## Contributing

Contributions are welcome! Please submit a PR or open an issue.

## License

Apache 2.0
