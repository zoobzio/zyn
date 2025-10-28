package zyn

import (
	"encoding/json"
	"testing"

	"github.com/zoobzio/sentinel"
)

// Test structs for schema generation.
type SimpleStruct struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ComplexStruct struct {
	Required string   `json:"required"`
	Optional *string  `json:"optional,omitempty"`
	List     []string `json:"list"`
	Ignored  string   `json:"-"`
}

type NestedStruct struct {
	Outer string `json:"outer"`
	Inner struct {
		Field string `json:"field"`
	} `json:"inner"`
}

func TestGenerateJSONSchema(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		schema := generateJSONSchema[SimpleStruct]()
		if schema == "" || schema == "{}" {
			t.Error("Expected non-empty schema")
		}

		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
			t.Errorf("Schema is not valid JSON: %v", err)
		}

		// Check basic structure
		if parsed["type"] != "object" {
			t.Errorf("Expected type=object, got %v", parsed["type"])
		}
		if parsed["properties"] == nil {
			t.Error("Expected properties field")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test that schema generation is consistent
		schema1 := generateJSONSchema[SimpleStruct]()
		schema2 := generateJSONSchema[SimpleStruct]()
		if schema1 != schema2 {
			t.Error("Schema generation is not deterministic")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test with complex nested types
		schema := generateJSONSchema[NestedStruct]()
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
			t.Errorf("Failed to parse nested schema: %v", err)
		}
	})
}

func TestBuildProperties(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		metadata := sentinel.Inspect[SimpleStruct]()
		props := buildProperties(metadata.Fields)

		if len(props) == 0 {
			t.Error("Expected properties to be built")
		}
		if props["name"] == nil {
			t.Error("Expected name property")
		}
		if props["count"] == nil {
			t.Error("Expected count property")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test with omitempty and ignored fields
		metadata := sentinel.Inspect[ComplexStruct]()
		props := buildProperties(metadata.Fields)

		if props["-"] != nil {
			t.Error("Fields with json:\"-\" should be skipped")
		}
		if props["required"] == nil {
			t.Error("Expected required field")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test properties work with schema generation
		schema := generateJSONSchema[ComplexStruct]()
		var parsed map[string]interface{}
		json.Unmarshal([]byte(schema), &parsed)
		props := parsed["properties"].(map[string]interface{})

		if len(props) == 0 {
			t.Error("Properties should be present in full schema")
		}
	})
}

func TestBuildRequiredFields(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		metadata := sentinel.Inspect[SimpleStruct]()
		required := buildRequiredFields(metadata.Fields)

		if len(required) == 0 {
			t.Error("Expected required fields")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test with omitempty - should not be in required
		metadata := sentinel.Inspect[ComplexStruct]()
		required := buildRequiredFields(metadata.Fields)

		hasOptional := false
		for _, field := range required {
			if field == "optional" {
				hasOptional = true
			}
		}
		if hasOptional {
			t.Error("Fields with omitempty should not be required")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test required fields appear in full schema
		schema := generateJSONSchema[ComplexStruct]()
		var parsed map[string]interface{}
		json.Unmarshal([]byte(schema), &parsed)
		required := parsed["required"].([]interface{})

		if len(required) == 0 {
			t.Error("Required fields should be present in schema")
		}
	})
}

func TestGetJSONFieldName(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		field := sentinel.FieldMetadata{
			Name: "TestField",
			Tags: map[string]string{"json": "test_field"},
		}
		name := getJSONFieldName(field)
		if name != "test_field" {
			t.Errorf("Expected 'test_field', got '%s'", name)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test with omitempty
		field := sentinel.FieldMetadata{
			Name: "TestField",
			Tags: map[string]string{"json": "test_field,omitempty"},
		}
		name := getJSONFieldName(field)
		if name != "test_field" {
			t.Errorf("Expected 'test_field' (without omitempty), got '%s'", name)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test field name extraction works with buildProperties
		field := sentinel.FieldMetadata{
			Name: "TestField",
			Tags: map[string]string{"json": "custom_name"},
			Type: "string",
		}
		props := buildProperties([]sentinel.FieldMetadata{field})
		if props["custom_name"] == nil {
			t.Error("Custom field name should appear in properties")
		}
	})
}

func TestHasOmitempty(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		field := sentinel.FieldMetadata{
			Tags: map[string]string{"json": "field,omitempty"},
		}
		if !hasOmitempty(field) {
			t.Error("Expected hasOmitempty to return true")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test without omitempty
		field := sentinel.FieldMetadata{
			Tags: map[string]string{"json": "field"},
		}
		if hasOmitempty(field) {
			t.Error("Expected hasOmitempty to return false")
		}

		// Test with no json tag
		fieldNoTag := sentinel.FieldMetadata{
			Tags: map[string]string{},
		}
		if hasOmitempty(fieldNoTag) {
			t.Error("Expected hasOmitempty to return false for field with no tag")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test hasOmitempty detection works with buildRequiredFields
		withOmit := sentinel.FieldMetadata{
			Name: "Optional",
			Tags: map[string]string{"json": "optional,omitempty"},
			Type: "string",
		}
		withoutOmit := sentinel.FieldMetadata{
			Name: "Required",
			Tags: map[string]string{"json": "required"},
			Type: "string",
		}
		required := buildRequiredFields([]sentinel.FieldMetadata{withOmit, withoutOmit})

		hasOptional := false
		hasRequired := false
		for _, name := range required {
			if name == "optional" {
				hasOptional = true
			}
			if name == "required" {
				hasRequired = true
			}
		}
		if hasOptional {
			t.Error("Optional field should not be in required list")
		}
		if !hasRequired {
			t.Error("Required field should be in required list")
		}
	})
}

func TestGoTypeToJSONType(t *testing.T) {
	tests := []struct {
		goType   string
		jsonType string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"int32", "integer"},
		{"int64", "integer"},
		{"float32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"[]string", "array"},
		{"[]int", "array"},
		{"map[string]string", "object"},
		{"CustomType", "object"},
	}

	t.Run("simple", func(t *testing.T) {
		result := goTypeToJSONType("string")
		if result != "string" {
			t.Errorf("Expected 'string', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Test all type conversions
		for _, tt := range tests {
			result := goTypeToJSONType(tt.goType)
			if result != tt.jsonType {
				t.Errorf("goTypeToJSONType(%s) = %s, want %s", tt.goType, result, tt.jsonType)
			}
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test type conversion works with buildProperties
		field := sentinel.FieldMetadata{
			Name: "Count",
			Tags: map[string]string{"json": "count"},
			Type: "int",
		}
		props := buildProperties([]sentinel.FieldMetadata{field})
		countProp := props["count"].(map[string]interface{})
		if countProp["type"] != "integer" {
			t.Errorf("Expected type=integer for int field, got %v", countProp["type"])
		}
	})
}
