package zyn

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/zoobzio/sentinel"
)

// mapTypeRegex matches map[K]V patterns and captures the value type.
var mapTypeRegex = regexp.MustCompile(`^map\[[^\]]+\](.+)$`)

// JSON Schema type constants.
const (
	jsonTypeObject  = "object"
	jsonTypeString  = "string"
	jsonTypeInteger = "integer"
	jsonTypeNumber  = "number"
	jsonTypeBoolean = "boolean"
	jsonTypeArray   = "array"
)

// Go type name constants for type detection.
const (
	goTypeString = "string"
	goTypeBool   = "bool"
)

// JSONSchema represents a JSON Schema object with full type safety.
// This is used to generate schemas for LLM response validation.
type JSONSchema struct {
	Type                    string                 `json:"-"` // handled in MarshalJSON
	Properties              map[string]*JSONSchema `json:"-"` // handled in MarshalJSON
	Items                   *JSONSchema            `json:"-"` // for arrays
	Required                []string               `json:"-"` // handled in MarshalJSON
	Description             string                 `json:"-"` // optional field description
	AdditionalProperties    *JSONSchema            `json:"-"` // for map value types
	DisallowAdditionalProps bool                   `json:"-"` // when true, additionalProperties: false
}

// MarshalJSON implements custom JSON marshaling to handle the additionalProperties
// field which can be either false (boolean) or a schema object in JSON Schema.
func (s *JSONSchema) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	if s.Type != "" {
		m["type"] = s.Type
	}
	if len(s.Properties) > 0 {
		m["properties"] = s.Properties
	}
	if s.Items != nil {
		m["items"] = s.Items
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if s.Description != "" {
		m["description"] = s.Description
	}

	// Handle additionalProperties: either false or a schema
	if s.DisallowAdditionalProps {
		m["additionalProperties"] = false
	} else if s.AdditionalProperties != nil {
		m["additionalProperties"] = s.AdditionalProperties
	}

	return json.Marshal(m)
}

// generateJSONSchema creates a proper JSON Schema from a Go type using sentinel.
// Uses Scan to recursively register nested types, then builds a complete schema.
// Returns an error if the schema cannot be marshaled to JSON.
func generateJSONSchema[T any]() (string, error) {
	// Use Scan to recursively register all nested types in the same module
	metadata := sentinel.Scan[T]()

	// Build the schema recursively
	schema := buildSchemaFromMetadata(metadata, true)

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to generate JSON schema: %w", err)
	}

	return string(jsonBytes), nil
}

// buildSchemaFromMetadata constructs a JSONSchema from sentinel metadata.
// isRoot indicates if this is the top-level schema (affects additionalProperties handling).
func buildSchemaFromMetadata(metadata sentinel.ModelMetadata, isRoot bool) *JSONSchema {
	schema := &JSONSchema{
		Type:                    jsonTypeObject,
		Properties:              make(map[string]*JSONSchema),
		DisallowAdditionalProps: isRoot, // root objects disallow extra properties
	}

	// Build a map of field name -> relationship for quick lookup
	relMap := make(map[string]sentinel.TypeRelationship)
	for _, rel := range metadata.Relationships {
		relMap[rel.Field] = rel
	}

	for _, field := range metadata.Fields {
		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue // Skip fields with json:"-"
		}

		// Build schema for this field
		fieldSchema := buildFieldSchema(field, relMap)

		// Add description if available
		if desc, ok := field.Tags["desc"]; ok {
			fieldSchema.Description = desc
		}

		schema.Properties[jsonName] = fieldSchema

		// Track required fields
		if !hasOmitempty(field) {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

// buildFieldSchema creates a JSONSchema for a single field.
func buildFieldSchema(field sentinel.FieldMetadata, relMap map[string]sentinel.TypeRelationship) *JSONSchema {
	typeStr := field.Type

	// Check if this field has a relationship (nested struct)
	if rel, hasRel := relMap[field.Name]; hasRel {
		return buildRelationshipSchema(rel)
	}

	// Handle primitive types and containers
	return buildPrimitiveSchema(typeStr)
}

// buildRelationshipSchema handles fields that reference other structs.
func buildRelationshipSchema(rel sentinel.TypeRelationship) *JSONSchema {
	switch rel.Kind {
	case sentinel.RelationshipReference, sentinel.RelationshipEmbedding:
		// Direct struct reference - look up and recurse
		if nested, found := sentinel.Lookup(rel.To); found {
			return buildSchemaFromMetadata(nested, false)
		}
		// Fallback if not found
		return &JSONSchema{Type: jsonTypeObject}

	case sentinel.RelationshipCollection:
		// Array of structs
		schema := &JSONSchema{Type: jsonTypeArray}
		if nested, found := sentinel.Lookup(rel.To); found {
			schema.Items = buildSchemaFromMetadata(nested, false)
		} else {
			schema.Items = &JSONSchema{Type: jsonTypeObject}
		}
		return schema

	case sentinel.RelationshipMap:
		// Map with struct values
		schema := &JSONSchema{Type: jsonTypeObject}
		if nested, found := sentinel.Lookup(rel.To); found {
			schema.AdditionalProperties = buildSchemaFromMetadata(nested, false)
		} else {
			schema.AdditionalProperties = &JSONSchema{Type: jsonTypeObject}
		}
		return schema

	default:
		return &JSONSchema{Type: jsonTypeObject}
	}
}

// buildPrimitiveSchema handles primitive types and primitive containers.
func buildPrimitiveSchema(typeStr string) *JSONSchema {
	// Check for array types: []T
	if strings.HasPrefix(typeStr, "[]") {
		elemType := strings.TrimPrefix(typeStr, "[]")
		return &JSONSchema{
			Type:  jsonTypeArray,
			Items: buildPrimitiveSchema(elemType),
		}
	}

	// Check for map types: map[K]V
	if strings.HasPrefix(typeStr, "map[") {
		valueType := parseMapValueType(typeStr)
		return &JSONSchema{
			Type:                 jsonTypeObject,
			AdditionalProperties: buildPrimitiveSchema(valueType),
		}
	}

	// Check for pointer types: *T
	if strings.HasPrefix(typeStr, "*") {
		elemType := strings.TrimPrefix(typeStr, "*")
		return buildPrimitiveSchema(elemType)
	}

	// Map Go primitive types to JSON Schema types
	return &JSONSchema{Type: goTypeToJSONType(typeStr)}
}

// parseMapValueType extracts the value type from a map type string.
// For example: "map[string]int" -> "int", "map[string][]string" -> "[]string".
func parseMapValueType(typeStr string) string {
	matches := mapTypeRegex.FindStringSubmatch(typeStr)
	if len(matches) == 2 {
		return matches[1]
	}
	return jsonTypeString // fallback
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

	// Default to fully lowercase field name (handles acronyms like "URL" -> "url")
	return strings.ToLower(field.Name)
}

// hasOmitempty checks if the json tag contains omitempty.
func hasOmitempty(field sentinel.FieldMetadata) bool {
	if jsonTag, ok := field.Tags["json"]; ok {
		return strings.Contains(jsonTag, "omitempty")
	}
	return false
}

// goTypeToJSONType maps Go primitive types to JSON Schema types.
func goTypeToJSONType(goType string) string {
	switch goType {
	case goTypeString:
		return jsonTypeString
	case goTypeBool:
		return jsonTypeBoolean
	}

	// Handle numeric types with prefix matching
	switch {
	case strings.HasPrefix(goType, "int"), strings.HasPrefix(goType, "uint"):
		return jsonTypeInteger
	case strings.HasPrefix(goType, "float"), strings.HasPrefix(goType, "complex"):
		return jsonTypeNumber
	default:
		// Unknown types default to object
		return jsonTypeObject
	}
}
