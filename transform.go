package prompterizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"maps"

	"github.com/samber/lo"
	"google.golang.org/genai"
)

type FieldParams struct {
	Name        string
	Type        genai.Type
	Format      *string
	Description string
	Aliases     []string
	IsRequired  bool
}

func MarshalResponseSchema(v any, descriptionVars map[string]string) (*genai.Schema, error) {
	var t reflect.Type
	schema := &genai.Schema{
		Type:       genai.TypeObject,
		Properties: map[string]*genai.Schema{},
	}

	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		t = v.(reflect.Type)
	} else {
		t = reflect.TypeOf(v)
	}

	// Get indirect value to handle pointers
	value := reflect.Indirect(reflect.ValueOf(v))

	for i := range t.NumField() {
		field := t.Field(i)

		// Add recursion for embedded structs
		if field.Anonymous {
			valueField := value.Field(i)
			embeddedSchema, err := MarshalResponseSchema(valueField.Interface(), descriptionVars)
			if err != nil {
				return nil, err
			}
			// Add all properties from the embedded schema to the parent schema
			maps.Copy(schema.Properties, embeddedSchema.Properties)

			// If the embedded struct has required fields, add them to the parent schema's required fields
			if len(embeddedSchema.Required) > 0 {
				schema.Required = append(schema.Required, embeddedSchema.Required...)
			}

			continue
		}

		fieldParams, err := parseFieldParams(field.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field params for %s: %w", field.Name, err)
		}
		if fieldParams == nil {
			continue
		}

		if fieldParams.IsRequired {
			schema.Required = append(schema.Required, fieldParams.Name)
		}

		description, err := renderDescription(fieldParams, descriptionVars)
		if err != nil {
			return nil, err
		}
		switch true {
		case field.Type.Kind() == reflect.Slice: // Arrays
			arraySchema := &genai.Schema{
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type:        fieldParams.Type,
					Description: description,
				},
			}

			if field.Type.Elem().Kind() == reflect.Struct { // Array of structs
				nestedSchema, err := MarshalResponseSchema(field.Type.Elem(), descriptionVars)
				if err != nil {
					return nil, err
				}

				arraySchema.Items = nestedSchema
			}

			schema.Properties[fieldParams.Name] = arraySchema

			continue
		case fieldParams.Type == genai.TypeObject: // Nested structs
			nestedSchema, err := MarshalResponseSchema(field.Type, descriptionVars)
			if err != nil {
				return nil, err
			}

			nestedSchema.Type = genai.TypeObject
			nestedSchema.Description = description

			schema.Properties[fieldParams.Name] = nestedSchema

			continue
		default:
			schema.Properties[fieldParams.Name] = &genai.Schema{
				Type:        fieldParams.Type,
				Description: description,
			}
		}

		if fieldParams.Format != nil {
			schema.Properties[fieldParams.Name].Format = *fieldParams.Format
		}
	}

	return schema, nil
}

func Unmarshal[T any](responseJson string) (T, error) {
	out := new(T)
	err := json.Unmarshal([]byte(responseJson), out)
	if err != nil {
		return *new(T), fmt.Errorf("unable to unmarshal prompt response '%s': %w", responseJson, err)
	}

	return *out, nil
}

func parseFieldParams(tag reflect.StructTag) (*FieldParams, error) {
	promptTag := tag.Get("prompt")
	if promptTag == "" {
		return nil, nil
	}

	promptTagParts := strings.Split(promptTag, ",")
	if len(promptTagParts) < 2 {
		return nil, errors.New("missing either prompt property name or type")
	}

	isRequired := lo.Contains(promptTagParts, "required")

	fieldType, err := toGenaiType(promptTagParts[1])
	if err != nil {
		return nil, err
	}

	description := tag.Get("prompt_description")

	fieldParams := &FieldParams{
		Name:        promptTagParts[0],
		Type:        fieldType,
		IsRequired:  isRequired,
		Description: description,
	}

	aliasesTag := tag.Get("prompt_aliases")
	if aliasesTag != "" {
		fieldParams.Aliases = strings.Split(aliasesTag, ",")
	}

	if fieldParams.Type == genai.TypeNumber {
		fieldParams.Format = lo.ToPtr("float")
	}

	return fieldParams, nil
}

func toGenaiType(propertyType string) (genai.Type, error) {
	switch propertyType {
	case "string":
		return genai.TypeString, nil
	case "bool":
		return genai.TypeBoolean, nil
	case "number":
		return genai.TypeNumber, nil
	case "integer":
		return genai.TypeInteger, nil
	case "array":
		return genai.TypeArray, nil
	case "object":
		return genai.TypeObject, nil
	default:
		return genai.TypeUnspecified, fmt.Errorf("unsupported property type %s", propertyType)
	}
}

func renderDescription(fieldParams *FieldParams, variables map[string]string) (string, error) {
	descriptionParts := []string{}

	missingVariables := []string{}
	if fieldParams.Description != "" {
		descriptionParts = append(descriptionParts, fieldParams.Description)
		matches := regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(descriptionParts[0], -1)
		for _, match := range matches {
			if value, ok := variables[match[1]]; ok {
				descriptionParts[0] = strings.ReplaceAll(descriptionParts[0], match[0], value)
			} else {
				missingVariables = append(missingVariables, match[1])
			}
		}
	}

	if len(fieldParams.Aliases) > 0 {
		formattedAliases := lo.Map(fieldParams.Aliases, func(alias string, _ int) string { return fmt.Sprintf("'%s'", strings.TrimSpace(alias)) })
		descriptionParts = append(descriptionParts, fmt.Sprintf("Also commonly reported as %s.", strings.Join(formattedAliases, ", ")))
	}
	description := strings.Join(descriptionParts, " ")

	if len(missingVariables) > 0 {
		return description, fmt.Errorf("missing variables in description: %s", strings.Join(missingVariables, ", "))
	}
	return description, nil
}
