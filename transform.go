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
	Enum        []string
	Description string
	Aliases     []string
	IsRequired  bool
}

func Unmarshal[T any](responseJson string) (T, error) {
	out := new(T)
	err := json.Unmarshal([]byte(responseJson), out)
	if err != nil {
		return *new(T), fmt.Errorf("unable to unmarshal prompt response '%s': %w", responseJson, err)
	}

	return *out, nil
}

func MarshalResponseSchema(v any, descriptionVars map[string]string) (*genai.Schema, error) {
	if v == nil {
		return nil, errors.New("input value for schema generation cannot be nil")
	}

	vType := reflect.TypeOf(v)
	if vType.Kind() == reflect.Pointer {
		vType = vType.Elem()
	}

	if vType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input value for schema generation must be a struct, got %s", vType.Kind())
	}

	return marshalType(reflect.TypeOf(v), genai.TypeObject, descriptionVars)
}

func marshalType(currentType reflect.Type, promptType genai.Type, descriptionVars map[string]string) (*genai.Schema, error) {
	switch currentType.Kind() {
	case reflect.Pointer:
		elementType := currentType.Elem()
		schema, err := marshalType(elementType, promptType, descriptionVars)
		if err != nil {
			return nil, err
		}
		schema.Nullable = lo.ToPtr(true)
		return schema, nil

	case reflect.Struct:
		if promptType != genai.TypeObject {
			return &genai.Schema{
				Type: promptType,
			}, nil
		}

		schema := &genai.Schema{
			Type:       genai.TypeObject,
			Properties: map[string]*genai.Schema{},
		}

		// Track property ordering as we process fields
		var propertyOrdering []string

		for i := 0; i < currentType.NumField(); i++ {
			field := currentType.Field(i)
			if !field.IsExported() { // Skip unexported fields
				continue
			}

			// Handle embedded structs
			if field.Anonymous {
				embeddedSchema, err := marshalType(field.Type, genai.TypeObject, descriptionVars)
				if err != nil {
					return nil, fmt.Errorf("error marshaling embedded field %s: %w", field.Name, err)
				}

				// Add embedded properties to ordering first
				if len(embeddedSchema.PropertyOrdering) > 0 {
					propertyOrdering = append(propertyOrdering, embeddedSchema.PropertyOrdering...)
				}
				maps.Copy(schema.Properties, embeddedSchema.Properties)
				if len(embeddedSchema.Required) > 0 {
					schema.Required = append(schema.Required, embeddedSchema.Required...)
				}
				continue
			}

			fieldParams, err := parseFieldParams(field.Tag)
			if err != nil {
				return nil, fmt.Errorf("failed to parse field params for %s: %w", field.Name, err)
			}
			if fieldParams == nil { // No "prompt" tag, skip this field
				continue
			}

			fieldSchema, err := marshalType(field.Type, fieldParams.Type, descriptionVars)
			if err != nil {
				return nil, fmt.Errorf("error marshaling property %s (Go field %s, type %s): %w", fieldParams.Name, field.Name, field.Type.String(), err)
			}

			if err := validateMarshaledFieldType(fieldSchema, fieldParams); err != nil {
				return nil, err
			}

			description, err := renderDescription(fieldParams, descriptionVars)
			if err != nil {
				return nil, fmt.Errorf("error rendering description for %s: %w", fieldParams.Name, err)
			}

			fieldSchema.Description = description
			fieldSchema.Format = lo.FromPtr(fieldParams.Format)
			fieldSchema.Enum = fieldParams.Enum

			schema.Properties[fieldParams.Name] = fieldSchema
			// Add property to ordering list
			propertyOrdering = append(propertyOrdering, fieldParams.Name)
			if fieldParams.IsRequired {
				schema.Required = append(schema.Required, fieldParams.Name)
			}
		}

		if len(schema.Required) > 0 {
			schema.Required = lo.Uniq(schema.Required)
		}
		// Set property ordering to maintain the order from the struct
		if len(propertyOrdering) > 0 {
			schema.PropertyOrdering = propertyOrdering
		}
		return schema, nil

	case reflect.Slice, reflect.Array:
		elemType := currentType.Elem()

		itemsSchema, err := marshalType(elemType, promptType, descriptionVars)
		if err != nil {
			return nil, fmt.Errorf("error marshaling array/slice items of type %s: %w", elemType.String(), err)
		}
		return &genai.Schema{Type: genai.TypeArray, Items: itemsSchema}, nil

	// Primitives
	case reflect.String:
		return &genai.Schema{Type: genai.TypeString}, nil
	case reflect.Bool:
		return &genai.Schema{Type: genai.TypeBoolean}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return &genai.Schema{Type: genai.TypeInteger}, nil
	case reflect.Float32, reflect.Float64:
		return &genai.Schema{Type: genai.TypeNumber}, nil

	default:
		return nil, fmt.Errorf("unsupported type kind for schema generation: %s (Go type: %s)", currentType.Kind(), currentType.String())
	}
}

func validateMarshaledFieldType(marshaledFieldSchema *genai.Schema, promptFieldParams *FieldParams) error {
	if marshaledFieldSchema.Type == genai.TypeArray {
		return validateMarshaledFieldType(marshaledFieldSchema.Items, promptFieldParams)
	}

	if marshaledFieldSchema.Type != genai.TypeUnspecified &&
		marshaledFieldSchema.Type == promptFieldParams.Type {
		return nil
	}

	return fmt.Errorf(
		"type mismatch for field '%s': Go type implies %s, but prompt tag specifies %s",
		promptFieldParams.Name, marshaledFieldSchema.Type, promptFieldParams.Type,
	)
}

func parseFieldParams(tag reflect.StructTag) (*FieldParams, error) {
	promptTag := tag.Get("prompt")
	if promptTag == "" {
		return nil, nil
	}

	promptTagParts := strings.Split(promptTag, ",")

	isRequired := lo.Contains(promptTagParts, "required")
	if isRequired {
		promptTagParts = lo.Reject(promptTagParts, func(p string, _ int) bool { return p == "required" })
	}

	if len(promptTagParts) < 2 {
		return nil, errors.New("missing either prompt property name or type")
	}

	fieldName := promptTagParts[0]
	fieldType, err := toGenaiType(promptTagParts[1])
	if err != nil {
		return nil, err
	}

	var explicitFormat string
	if len(promptTagParts) > 2 {
		explicitFormat = promptTagParts[2]
	}

	fieldParams := &FieldParams{
		Name:        fieldName,
		Type:        fieldType,
		Enum:        parseCommaSeparated(tag.Get("prompt_enum")),
		Aliases:     parseCommaSeparated(tag.Get("prompt_aliases")),
		IsRequired:  isRequired,
		Description: tag.Get("prompt_description"),
	}

	switch {
	case explicitFormat != "":
		fieldParams.Format = &explicitFormat
	case len(fieldParams.Enum) > 0:
		fieldParams.Format = lo.ToPtr("enum")
	case fieldType == genai.TypeNumber:
		fieldParams.Format = lo.ToPtr("float")
	}

	return fieldParams, nil
}

func toGenaiType(promptFieldType string) (genai.Type, error) {
	switch promptFieldType {
	case "string":
		return genai.TypeString, nil
	case "bool":
		return genai.TypeBoolean, nil
	case "number":
		return genai.TypeNumber, nil
	case "integer":
		return genai.TypeInteger, nil
	case "object":
		return genai.TypeObject, nil
	default:
		return genai.TypeUnspecified, fmt.Errorf("unsupported field type %s", promptFieldType)
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

func parseCommaSeparated(tag string) []string {
	if tag == "" {
		return nil
	}

	parts := strings.Split(tag, ",")
	return lo.Map(parts, func(part string, _ int) string { return strings.TrimSpace(part) })
}
