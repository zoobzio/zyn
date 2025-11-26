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

type NestedInner struct {
	Value float64 `json:"value"`
	Label string  `json:"label"`
}

type NestedOuter struct {
	Name  string      `json:"name"`
	Inner NestedInner `json:"inner"`
}

type WithArrayOfStructs struct {
	Items []NestedInner `json:"items"`
}

type WithMap struct {
	Data map[string]string `json:"data"`
}

type WithMapOfStructs struct {
	Records map[string]NestedInner `json:"records"`
}

func TestGenerateJSONSchema(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		schema, err := generateJSONSchema[SimpleStruct]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		if err := json.Unmarshal([]byte(schema), &rawParsed); err != nil {
			t.Fatalf("schema is not valid JSON: %v", err)
		}

		if rawParsed["type"] != "object" {
			t.Errorf("expected type=object, got %v", rawParsed["type"])
		}

		// Check properties exist
		props := rawParsed["properties"].(map[string]any)

		if props["name"] == nil {
			t.Error("expected 'name' property")
		}
		if props["count"] == nil {
			t.Error("expected 'count' property")
		}
	})

	t.Run("complex struct with omitempty", func(t *testing.T) {
		schema, err := generateJSONSchema[ComplexStruct]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)

		// Check required fields - optional should not be required
		required := rawParsed["required"].([]any)
		hasOptional := false
		hasRequired := false
		for _, r := range required {
			if r == "optional" {
				hasOptional = true
			}
			if r == "required" {
				hasRequired = true
			}
		}
		if hasOptional {
			t.Error("'optional' should not be in required list")
		}
		if !hasRequired {
			t.Error("'required' should be in required list")
		}

		// Check ignored field is not present
		props := rawParsed["properties"].(map[string]any)
		if props["-"] != nil {
			t.Error("ignored field should not appear")
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		schema, err := generateJSONSchema[NestedOuter]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)
		props := rawParsed["properties"].(map[string]any)

		// Check inner has nested properties
		inner := props["inner"].(map[string]any)
		if inner["type"] != "object" {
			t.Errorf("expected inner.type=object, got %v", inner["type"])
		}

		innerProps := inner["properties"].(map[string]any)
		if innerProps["value"] == nil {
			t.Error("expected inner to have 'value' property")
		}
		if innerProps["label"] == nil {
			t.Error("expected inner to have 'label' property")
		}
	})

	t.Run("array of primitives", func(t *testing.T) {
		schema, err := generateJSONSchema[ComplexStruct]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)
		props := rawParsed["properties"].(map[string]any)

		list := props["list"].(map[string]any)
		if list["type"] != "array" {
			t.Errorf("expected list.type=array, got %v", list["type"])
		}

		items := list["items"].(map[string]any)
		if items["type"] != "string" {
			t.Errorf("expected list.items.type=string, got %v", items["type"])
		}
	})

	t.Run("array of structs", func(t *testing.T) {
		schema, err := generateJSONSchema[WithArrayOfStructs]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)
		props := rawParsed["properties"].(map[string]any)

		items := props["items"].(map[string]any)
		if items["type"] != "array" {
			t.Errorf("expected items.type=array, got %v", items["type"])
		}

		arrayItems := items["items"].(map[string]any)
		if arrayItems["type"] != "object" {
			t.Errorf("expected items.items.type=object, got %v", arrayItems["type"])
		}

		// Check nested struct properties are present
		itemProps := arrayItems["properties"].(map[string]any)
		if itemProps["value"] == nil {
			t.Error("expected array item to have 'value' property")
		}
	})

	t.Run("map of primitives", func(t *testing.T) {
		schema, err := generateJSONSchema[WithMap]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)
		props := rawParsed["properties"].(map[string]any)

		data := props["data"].(map[string]any)
		if data["type"] != "object" {
			t.Errorf("expected data.type=object, got %v", data["type"])
		}

		addProps := data["additionalProperties"].(map[string]any)
		if addProps["type"] != "string" {
			t.Errorf("expected data.additionalProperties.type=string, got %v", addProps["type"])
		}
	})

	t.Run("map of structs", func(t *testing.T) {
		schema, err := generateJSONSchema[WithMapOfStructs]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)
		props := rawParsed["properties"].(map[string]any)

		records := props["records"].(map[string]any)
		if records["type"] != "object" {
			t.Errorf("expected records.type=object, got %v", records["type"])
		}

		addProps := records["additionalProperties"].(map[string]any)
		if addProps["type"] != "object" {
			t.Errorf("expected records.additionalProperties.type=object, got %v", addProps["type"])
		}

		// Check nested struct properties
		nestedProps := addProps["properties"].(map[string]any)
		if nestedProps["value"] == nil {
			t.Error("expected map value struct to have 'value' property")
		}
	})

	t.Run("additionalProperties false at root", func(t *testing.T) {
		schema, err := generateJSONSchema[SimpleStruct]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}

		var rawParsed map[string]any
		json.Unmarshal([]byte(schema), &rawParsed)

		addProps := rawParsed["additionalProperties"]
		if addProps != false {
			t.Errorf("expected root additionalProperties=false, got %v", addProps)
		}
	})

	t.Run("deterministic output", func(t *testing.T) {
		schema1, _ := generateJSONSchema[SimpleStruct]()
		schema2, _ := generateJSONSchema[SimpleStruct]()
		if schema1 != schema2 {
			t.Error("schema generation should be deterministic")
		}
	})
}

func TestGetJSONFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    FieldMetadataInput
		expected string
	}{
		{
			name:     "simple json tag",
			field:    FieldMetadataInput{Name: "TestField", JSONTag: "test_field"},
			expected: "test_field",
		},
		{
			name:     "json tag with omitempty",
			field:    FieldMetadataInput{Name: "TestField", JSONTag: "test_field,omitempty"},
			expected: "test_field",
		},
		{
			name:     "no json tag",
			field:    FieldMetadataInput{Name: "TestField", JSONTag: ""},
			expected: "testfield",
		},
		{
			name:     "json skip tag",
			field:    FieldMetadataInput{Name: "TestField", JSONTag: "-"},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := tt.field.toSentinelField()
			result := getJSONFieldName(field)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHasOmitempty(t *testing.T) {
	tests := []struct {
		name     string
		jsonTag  string
		expected bool
	}{
		{"with omitempty", "field,omitempty", true},
		{"without omitempty", "field", false},
		{"empty tag", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := FieldMetadataInput{Name: "Test", JSONTag: tt.jsonTag}.toSentinelField()
			result := hasOmitempty(field)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
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
		{"uint", "integer"},
		{"float32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"CustomType", "object"},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			result := goTypeToJSONType(tt.goType)
			if result != tt.jsonType {
				t.Errorf("goTypeToJSONType(%q) = %q, want %q", tt.goType, result, tt.jsonType)
			}
		})
	}
}

func TestParseMapValueType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"map[string]string", "string"},
		{"map[string]int", "int"},
		{"map[string][]string", "[]string"},
		{"map[int]bool", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseMapValueType(tt.input)
			if result != tt.expected {
				t.Errorf("parseMapValueType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper for creating test sentinel field metadata.
type FieldMetadataInput struct {
	Name    string
	JSONTag string
}

func (f FieldMetadataInput) toSentinelField() sentinel.FieldMetadata {
	tags := make(map[string]string)
	if f.JSONTag != "" {
		tags["json"] = f.JSONTag
	}
	return sentinel.FieldMetadata{
		Name: f.Name,
		Tags: tags,
	}
}
