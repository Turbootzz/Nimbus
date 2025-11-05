package models

import (
	"encoding/json"
	"testing"
)

func TestNullableString_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantSet  bool
		wantVal  *string
	}{
		{
			name:     "Explicit null value",
			jsonData: `{"field": null}`,
			wantSet:  true,
			wantVal:  nil,
		},
		{
			name:     "String value",
			jsonData: `{"field": "test"}`,
			wantSet:  true,
			wantVal:  stringPtr("test"),
		},
		{
			name:     "Empty string value",
			jsonData: `{"field": ""}`,
			wantSet:  true,
			wantVal:  stringPtr(""),
		},
	}

	type testStruct struct {
		Field NullableString `json:"field"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result testStruct
			err := json.Unmarshal([]byte(tt.jsonData), &result)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if result.Field.Set != tt.wantSet {
				t.Errorf("Set = %v, want %v", result.Field.Set, tt.wantSet)
			}

			if (result.Field.Value == nil) != (tt.wantVal == nil) {
				t.Errorf("Value nil mismatch: got %v, want %v", result.Field.Value, tt.wantVal)
			}

			if result.Field.Value != nil && tt.wantVal != nil {
				if *result.Field.Value != *tt.wantVal {
					t.Errorf("Value = %v, want %v", *result.Field.Value, *tt.wantVal)
				}
			}
		})
	}
}

func TestNullableString_UnmarshalJSON_OmittedField(t *testing.T) {
	type testStruct struct {
		Field NullableString `json:"field"`
	}

	// When field is omitted from JSON, Set should be false
	jsonData := `{}`
	var result testStruct
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Field.Set {
		t.Errorf("Omitted field: Set = true, want false")
	}
	if result.Field.Value != nil {
		t.Errorf("Omitted field: Value = %v, want nil", result.Field.Value)
	}
}

func TestNullableString_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		field    NullableString
		wantJSON string
	}{
		{
			name:     "Null value",
			field:    NullableString{Value: nil, Set: true},
			wantJSON: `{"field":null}`,
		},
		{
			name:     "String value",
			field:    NullableString{Value: stringPtr("test"), Set: true},
			wantJSON: `{"field":"test"}`,
		},
		{
			name:     "Empty string",
			field:    NullableString{Value: stringPtr(""), Set: true},
			wantJSON: `{"field":""}`,
		},
	}

	type testStruct struct {
		Field NullableString `json:"field"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := testStruct{Field: tt.field}
			jsonData, err := json.Marshal(data)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if string(jsonData) != tt.wantJSON {
				t.Errorf("Marshal = %s, want %s", string(jsonData), tt.wantJSON)
			}
		})
	}
}

func TestNullableString_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name:     "Null value round trip",
			jsonData: `{"background":null,"accent":null}`,
		},
		{
			name:     "String values round trip",
			jsonData: `{"background":"https://example.com/bg.jpg","accent":"#3B82F6"}`,
		},
		{
			name:     "Mixed values round trip",
			jsonData: `{"background":"https://example.com/bg.jpg","accent":null}`,
		},
	}

	type testStruct struct {
		Background NullableString `json:"background"`
		Accent     NullableString `json:"accent"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Unmarshal
			var result testStruct
			err := json.Unmarshal([]byte(tt.jsonData), &result)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Marshal back
			jsonData, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			if string(jsonData) != tt.jsonData {
				t.Errorf("Round trip failed: got %s, want %s", string(jsonData), tt.jsonData)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
