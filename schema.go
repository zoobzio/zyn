package zyn

import (
	"encoding/json"
	"strings"

	"github.com/zoobzio/sentinel"
)

// generateJSONSchema creates a proper JSON Schema from a Go type using sentinel.
func generateJSONSchema[T any]() string {
	// Use sentinel to extract metadata for struct types
	metadata := sentinel.Inspect[T]()

	// Build JSON Schema object
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           buildProperties(metadata.Fields),
		"required":             buildRequiredFields(metadata.Fields),
		"additionalProperties": false,
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		// Fallback to simple representation
		return "{}"
	}

	return string(jsonBytes)
}

// buildProperties converts field metadata to JSON Schema properties.
func buildProperties(fields []sentinel.FieldMetadata) map[string]interface{} {
	properties := make(map[string]interface{})

	for _, field := range fields {
		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue // Skip fields with json:"-"
		}

		properties[jsonName] = map[string]interface{}{
			"type": goTypeToJSONType(field.Type),
		}

		// Add description if available
		if desc, ok := field.Tags["desc"]; ok {
			if prop, ok := properties[jsonName].(map[string]interface{}); ok {
				prop["description"] = desc
			}
		}
	}

	return properties
}

// buildRequiredFields determines which fields are required.
func buildRequiredFields(fields []sentinel.FieldMetadata) []string {
	var required []string

	for _, field := range fields {
		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue
		}

		// Field is required unless it has omitempty in json tag
		if !hasOmitempty(field) {
			required = append(required, jsonName)
		}
	}

	return required
}

// getJSONFieldName extracts the JSON field name from metadata.
func getJSONFieldName(field sentinel.FieldMetadata) string {
	if jsonTag, ok := field.Tags["json"]; ok {
		// Handle "name,omitempty" format
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Default to lowercase field name
	return strings.ToLower(field.Name[:1]) + field.Name[1:]
}

// hasOmitempty checks if the json tag contains omitempty.
func hasOmitempty(field sentinel.FieldMetadata) bool {
	if jsonTag, ok := field.Tags["json"]; ok {
		return strings.Contains(jsonTag, "omitempty")
	}
	return false
}

// goTypeToJSONType maps Go types to JSON Schema types.
func goTypeToJSONType(goType string) string {
	// Handle common types
	switch {
	case strings.HasPrefix(goType, "string"):
		return "string"
	case strings.HasPrefix(goType, "int"), strings.HasPrefix(goType, "uint"):
		return "integer"
	case strings.HasPrefix(goType, "float"), strings.HasPrefix(goType, "complex"):
		return "number"
	case strings.HasPrefix(goType, "bool"):
		return "boolean"
	case strings.HasPrefix(goType, "[]"):
		return "array"
	case strings.HasPrefix(goType, "map["):
		return "object"
	default:
		// For custom types, default to object
		return "object"
	}
}
