package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBooleanValueFalseAndExcludedContexts(t *testing.T) {
	for _, schema := range []any{
		false, // Named true roots have a separate boolean_root_test.go contract.
		map[string]any{"properties": map[string]any{"v": false}, "required": []string{"v"}},
		map[string]any{"properties": map[string]any{"unused": false}},
		map[string]any{"type": "array", "items": false},
		map[string]any{"properties": map[string]any{"v": map[string]any{"oneOf": []any{true, map[string]any{"type": "string"}}}}},
		map[string]any{"allOf": []any{map[string]any{"properties": map[string]any{"v": true}}, map[string]any{"properties": map[string]any{"v": map[string]any{"type": "string"}}}}},
	} {
		for _, checked := range []bool{false, true} {
			r := tupleRequest(t, schema)
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if value, ok := schema.(bool); ok && !value {
				if err != nil || !strings.Contains(source(t, got, "Probe"), "sealed class Probe private constructor()") {
					t.Fatal("original root false role did not graduate", err)
				}
				continue
			}
			// Preserve the exactoriginal required/unused literal-falsefield
			// controls as85 positivegraduations; othercontexts stayclosed.
			if reflect.DeepEqual(schema, map[string]any{"properties": map[string]any{"v": false}, "required": []string{"v"}}) || reflect.DeepEqual(schema, map[string]any{"properties": map[string]any{"unused": false}}) {
				if err != nil || !strings.Contains(source(t, got, "Probe"), "sealed class FalseValue private constructor()") {
					t.Fatal("literalfalsefield graduation", schema, err)
				}
				continue
			}
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("excluded source accepted/partial", schema, checked, got, err)
			}
		}
	}
	r := tupleRequest(t, map[string]any{"properties": map[string]any{"v": true}, "required": []string{"v"}})
	r.Document = []byte(strings.Replace(string(r.Document), "3.1.0", "3.0.3", 1))
	if got, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("3.0 Boolean broadened", got, err)
	}
}

func TestBooleanValueRootAndInternalReferenceExclusions(t *testing.T) {
	for _, input := range []string{
		`{"openapi":"3.1.0","components":{"schemas":{"A":{"properties":{"v":true}},"B":{"properties":{"v":{"$ref":"#/components/schemas/A/properties/v"}}}}}}`,
		// Direct 3.1 true JSON operation roots graduate through boolean_http_test.go.
	} {
		got, err := core.GenerateWithDiagnostics(request([]byte(input), ""))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("excluded source admitted", got, err)
		}
	}
	for _, schema := range []any{
		map[string]any{"properties": map[string]any{"v": false}},
		map[string]any{"properties": map[string]any{"v": map[string]any{"type": "array", "items": false}}},
	} {
		got, err := core.GenerateWithDiagnostics(tupleRequest(t, schema))
		if reflect.DeepEqual(schema, map[string]any{"properties": map[string]any{"v": false}}) {
			if err != nil || !strings.Contains(source(t, got, "Probe"), "AbsentSerializer") {
				t.Fatal("originaloptionalFalsefield graduation", err)
			}
			continue
		}
		if reflect.DeepEqual(schema, map[string]any{"properties": map[string]any{"v": map[string]any{"type": "array", "items": false}}}) {
			text := source(t, got, "Probe")
			if err != nil || !strings.Contains(text, "List<org.example.models.models.Probe.FalseValue>") || strings.Contains(text, "AbsentSerializer") {
				t.Fatal("original literal array field graduation", err, text)
			}
			continue
		}
		ds, ok := err.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != "E_SCHEMA_INVALID" || ds[0].Pointer == nil || !strings.HasPrefix(*ds[0].Pointer, "/components/schemas/Probe/properties/v") || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("false original source drift", got, err)
		}
	}
}

func TestBooleanValueOpaqueMetadataAndSourceWarning(t *testing.T) {
	for _, checked := range []bool{false, true} {
		r := tupleRequest(t, map[string]any{
			"properties": map[string]any{"v": true},
			"required":   []string{"v"},
			"examples":   []any{map[string]any{"properties": map[string]any{"ignoredTrue": true, "ignoredFalse": false}, "items": false}},
		})
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil || len(got.Warnings) != 1 || got.Warnings[0].Code != "W_BOOLEAN_SCHEMA_PROJECTION" || got.Warnings[0].Pointer == nil || *got.Warnings[0].Pointer != "/components/schemas/Probe/properties/v" {
			t.Fatal("metadata payload treated as schema", got, err)
		}
	}
}

func TestBooleanValueIndependentBudgets(t *testing.T) {
	for _, levels := range []int{48, 49} {
		var leaf any = true
		for i := 1; i < levels; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		r := tupleRequest(t, map[string]any{"properties": map[string]any{"v": leaf}, "required": []string{"v"}})
		got, err := core.GenerateWithDiagnostics(r)
		if levels == 48 && err != nil || levels == 49 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(levels, err)
		}
	}
	for _, edges := range []int{16, 17} {
		var leaf any = true
		for i := 0; i < edges; i++ {
			leaf = map[string]any{"type": "array", "minItems": 1, "items": leaf}
		}
		r := tupleRequest(t, map[string]any{"properties": map[string]any{"v": leaf}, "required": []string{"v"}})
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(edges, err)
		}
	}
	for _, width := range []int{240, 241} {
		props := map[string]any{}
		required := []string{}
		for i := 0; i < width; i++ {
			key := strings.Repeat("x", i+1)
			props[key] = true
			required = append(required, key)
		}
		r := tupleRequest(t, map[string]any{"properties": props, "required": required})
		got, err := core.GenerateWithDiagnostics(r)
		if width == 240 && err != nil || width == 241 && (err == nil || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, err)
		}
	}
}
