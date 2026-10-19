package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

// A document that mixes every recent admission in one place must be a fixed
// point: generating twice, with every option combination, produces byte-equal
// artifacts. Ordering or map-iteration dependence would show up here rather than
// in a single-feature test.
func TestMixedFeatureDocumentIsAFixedPoint(t *testing.T) {
	doc, err := json.Marshal(map[string]any{
		"openapi": "3.1.0",
		"paths": map[string]any{"/r": map[string]any{"post": map[string]any{
			"operationId": "submit",
			"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{
				"type": "object", "required": []string{"id"},
				"properties": map[string]any{
					"id":   map[string]any{"type": "string", "readOnly": true},
					"blob": map[string]any{"type": "string", "format": "byte", "default": []int{1}},
				},
			}}}},
			"responses": map[string]any{"204": map[string]any{}},
		}}},
		"components": map[string]any{"schemas": map[string]any{
			"Shape": map[string]any{
				"oneOf": []any{
					map[string]any{"type": "object", "required": []string{"kind"}, "properties": map[string]any{"kind": map[string]any{"type": "string", "const": "circle"}}},
				},
				"discriminator": map[string]any{"propertyName": "kind"},
			},
			"Target": map[string]any{"type": "object", "required": []string{"z"}, "properties": map[string]any{"z": map[string]any{"type": "string"}}},
			"Wrap":   map[string]any{"$ref": "#/components/schemas/Target", "default": map[string]any{"z": "1"}},
			"Blob":   map[string]any{"type": "string", "format": "byte", "default": []int{-128, 127}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			first := request(doc, "Dto")
			first.Options.Validation, first.Options.ModelStyle = validation, style
			a, err := core.GenerateWithDiagnostics(first)
			if err != nil {
				t.Fatalf("validation=%v style=%v: %v", validation, style, err)
			}
			second := request(doc, "Dto")
			second.Options.Validation, second.Options.ModelStyle = validation, style
			b, err := core.GenerateWithDiagnostics(second)
			if err != nil {
				t.Fatalf("second run (validation=%v style=%v): %v", validation, style, err)
			}
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("mixed document is not a fixed point (validation=%v style=%v)", validation, style)
			}
			filesOnly, err := core.Generate(second)
			if err != nil || !reflect.DeepEqual(filesOnly, a.Artifacts) {
				t.Fatalf("files-only and report APIs disagree (validation=%v style=%v)", validation, style)
			}
		}
	}
}

// A rejected document must not leave a partial result, even when the failure
// happens after several shapes were already planned successfully.
func TestRejectionAfterPartialPlanningLeavesNoResult(t *testing.T) {
	doc, err := json.Marshal(map[string]any{
		"openapi": "3.1.0",
		"components": map[string]any{"schemas": map[string]any{
			// Two shapes plan fine before the bad one is reached.
			"Good":   map[string]any{"type": "object", "required": []string{"z"}, "properties": map[string]any{"z": map[string]any{"type": "string"}}, "default": map[string]any{"z": "1"}},
			"AlsoOk": map[string]any{"type": "string", "format": "byte", "default": []int{1}},
			// base64 is not round-trippable, so this must fail the whole document.
			"Bad": map[string]any{"type": "string", "format": "byte", "default": "AAEC"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, validation := range []bool{false, true} {
		r := request(doc, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil {
			t.Fatalf("validation=%v: expected rejection", validation)
		}
		if len(got.Artifacts.Files) != 0 {
			t.Fatalf("validation=%v: rejection left %d partial files", validation, len(got.Artifacts.Files))
		}
		if len(got.Warnings) != 0 {
			t.Fatalf("validation=%v: rejection left warnings behind", validation)
		}
		if _, err := core.Generate(r); err == nil {
			t.Fatalf("validation=%v: files-only API accepted the rejected document", validation)
		}
	}
}
