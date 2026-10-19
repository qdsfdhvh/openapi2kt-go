package core_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Interaction case: a read-only field that also declares a default and is
// `required` must still be Patch.Absent. The default belongs to the response, so
// the request must not pre-fill the value; this asserts the projection wins over
// both the default and the required flag rather than being silently overridden
// by whichever rule runs last.
func TestReadOnlyWinsOverDefaultAndRequired(t *testing.T) {
	doc := []byte(`{"openapi":"3.1.0","paths":{"/r":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["id"],"properties":{"id":{"type":"string","readOnly":true,"default":"server"}}}}}},"responses":{"204":{}}}}}}`)
	for _, validation := range []bool{false, true} {
		r := request(doc, "Dto")
		r.Options.Validation = validation
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(validation, err)
		}
		text := source(t, got, "SubmitInputDto")
		if !strings.Contains(text, "public val id: org.example.models.models.Patch<kotlin.String> = org.example.models.models.Patch.Absent,") {
			t.Fatalf("readOnly+default+required did not project (validation=%v):\n%s", validation, text)
		}
		if strings.Contains(text, `"server"`) {
			t.Fatalf("the response default leaked into the request model:\n%s", text)
		}
	}
}

// Interaction case: a read-only member nested inside an inline request body keeps
// its own projection, and the enclosing model still exposes it as an object.
func TestReadOnlyProjectsInsideNestedRequestObjects(t *testing.T) {
	doc := []byte(`{"openapi":"3.1.0","paths":{"/r":{"post":{"operationId":"submit","requestBody":{"required":true,"content":{"application/json":{"schema":{"type":"object","required":["inner"],"properties":{"inner":{"type":"object","required":["id","name"],"properties":{"id":{"type":"string","readOnly":true},"name":{"type":"string"}}}}}}}},"responses":{"204":{}}}}}}`)
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "SubmitInputDto")
	if !strings.Contains(text, "public val id: org.example.models.models.Patch<kotlin.String> = org.example.models.models.Patch.Absent,") {
		t.Fatalf("nested readOnly member lost its projection:\n%s", text)
	}
	if !strings.Contains(text, "public val name: kotlin.String,") {
		t.Fatalf("nested writable member changed:\n%s", text)
	}
	// The enclosing member itself is not read-only, so it stays a plain object.
	if !strings.Contains(text, "public val inner:") || strings.Contains(text, "val inner: org.example.models.models.Patch<") {
		t.Fatalf("enclosing member was wrongly projected:\n%s", text)
	}
}

// Every admitted component-root shape in one document. They share the schema
// catalog and the factory planner, so a defect in one can corrupt another's
// declaration lookup or duplicate a helper.
func TestAllAdmittedComponentRootShapesCoexist(t *testing.T) {
	doc, err := json.Marshal(map[string]any{
		"openapi": "3.1.0",
		"components": map[string]any{"schemas": map[string]any{
			"Obj":      map[string]any{"type": "object", "required": []string{"z"}, "properties": map[string]any{"z": map[string]any{"type": "string"}}, "default": map[string]any{"z": "1"}},
			"Scalar":   map[string]any{"type": "string", "default": "x"},
			"Enum":     map[string]any{"type": "string", "enum": []string{"a"}, "default": "a"},
			"Temporal": map[string]any{"type": "string", "format": "date", "default": "2024-02-29"},
			"Bytes":    map[string]any{"type": "string", "format": "byte", "default": []int{-128, 127}},
			"List":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "default": []string{"x"}},
			"Map":      map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "integer"}, "default": map[string]any{"k": 1}},
			"Native":   map[string]any{"type": "integer", "format": "int32", "enum": []int{1, 2}, "default": 1},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, validation := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(doc, "Dto")
			r.Options.Validation, r.Options.ModelStyle = validation, style
			got, generateErr := core.GenerateWithDiagnostics(r)
			if generateErr != nil {
				t.Fatalf("validation=%v style=%v: %v", validation, style, generateErr)
			}
			// Each admitted root owns exactly one factory, and no helper is emitted twice.
			seen := map[string]int{}
			for _, file := range got.Artifacts.Files {
				for _, line := range strings.Split(file.Content, "\n") {
					if strings.Contains(line, "public fun `") && strings.Contains(line, " default`():") {
						seen[file.Path]++
					}
					if strings.Contains(line, "private fun `openapi-kt component default") {
						seen[file.Path+"/helper"]++
					}
				}
			}
			for path, count := range seen {
				if count > 1 && !strings.HasSuffix(path, "/helper") {
					t.Fatalf("duplicate factory in %s (validation=%v style=%v)", path, validation, style)
				}
				if strings.HasSuffix(path, "/helper") && count > 1 {
					t.Fatalf("duplicate decode helper in %s (validation=%v style=%v)", strings.TrimSuffix(path, "/helper"), validation, style)
				}
			}
			if len(seen) < 8 {
				t.Fatalf("expected a factory for all eight roots, saw %d (validation=%v style=%v)", len(seen), validation, style)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("multi-root document generation drifted", err)
			}
		}
	}
}

// Budget interaction: deep nesting combined with a default must fail at the
// nesting boundary, not be admitted because the default walk has its own budget.
func TestDeepNestingWithDefaultFailsAtTheDepthBoundary(t *testing.T) {
	body := map[string]any{"type": "string", "default": "x"}
	for i := 0; i < 46; i++ {
		body = map[string]any{"type": "array", "items": body}
	}
	doc, err := json.Marshal(map[string]any{
		"openapi":    "3.1.0",
		"components": map[string]any{"schemas": map[string]any{"Deep": body}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	var ds core.Diagnostics
	if !errors.As(err, &ds) || len(ds) == 0 {
		t.Fatalf("deep nesting with a default was admitted: %v", err)
	}
	if ds[0].Code != "E_SCHEMA_UNSUPPORTED" && ds[0].Code != "E_SCHEMA_DEPTH" {
		t.Fatalf("unexpected code %s", ds[0].Code)
	}
	if ds[0].Pointer == nil || !strings.Contains(*ds[0].Pointer, "/items/") {
		t.Fatalf("depth rejection did not name the nesting site: %v", ds[0].Pointer)
	}
	if len(got.Artifacts.Files) != 0 {
		t.Fatal("rejected deep document emitted files")
	}
}
