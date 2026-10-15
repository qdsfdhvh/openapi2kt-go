package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNamedNativeEnumsKeepNearestNullableAndOriginalBudget(t *testing.T) {
	for _, n := range []int{44, 45} {
		parts := []string{`"Code":{"type":"integer","enum":[1,2]}`}
		for i := range n {
			target := "Code"
			if i > 0 {
				target = fmt.Sprintf("A%d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"A%d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		parts = append(parts, fmt.Sprintf(`"Source":{"properties":{"hop":{"$ref":"#/components/schemas/A%d","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"},"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["value"]}`, n-1))
		r := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 44) {
			t.Fatal(n, e)
		}
		if e == nil {
			if !strings.Contains(source(t, got, "BridgeDto"), "typealias BridgeDto = kotlin.Int?") || strings.Contains(source(t, got, "ConsumerDto"), "value: kotlin.Int?") {
				t.Fatal("nearest nullable lost")
			}
		} else if e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(e)
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := `{"$ref":"#/components/schemas/Code"}`
		for range depth {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(schemas(`{"Code":{"type":"integer","enum":[1,2]},"Rows":{"$ref":"#/components/schemas/Source/properties/rows"},"Source":{"properties":{"rows":`+leaf+`}}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 16) {
			t.Fatal(depth, e)
		}
		if e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(e)
		}
	}
}

func TestNamedNativeEnumSourcesKeepExcludedAnnotationsAndProjectionChecks(t *testing.T) {
	for _, code := range []string{`{"type":"integer","enum":[1,2],"nullable":true}`, `{"type":"integer","enum":[1,2],"x-native-type":"bigint"}`, `{"type":"integer","enum":[1,2],"x-kotlin-value-class":true}`} {
		r := request(schemas(`{"Code":`+code+`,"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Code"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if code == `{"type":"integer","enum":[1,2],"nullable":true}` {
			// Preserve the original rejected named/internal bridge input;
			// nullable source authority now admits its checked scalar backing.
			if e != nil || !strings.Contains(source(t, got, "BridgeDto"), "typealias BridgeDto = kotlin.Int?") || !strings.Contains(source(t, got, "SourceDto"), "value == 1 || value == 2") {
				t.Fatal(got, e)
			}
			r.Options.Validation = false
			if off, err := core.GenerateWithDiagnostics(r); err == nil || !reflect.DeepEqual(off, core.GenerationReport{}) {
				t.Fatal(off, err)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(code, e)
		}
	}
	input := schemas(`{"Code":{"type":"integer","enum":[1,2]},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Code"}}},"Discarded":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/hop"}}},{"properties":{"value":{"type":"integer"}}}]}}`)
	r := request(input, "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/enum") || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal(e)
	}
}
