package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"testing"
)

func wideSignatureSchema(width, depth int) map[string]any {
	var leaf any = map[string]any{"type": "integer", "minimum": 1}
	for range depth {
		leaf = map[string]any{"type": "array", "items": leaf}
	}
	fields := map[string]any{}
	for i := range width {
		fields[fmt.Sprintf("p%d", i)] = leaf
	}
	return map[string]any{"type": "object", "properties": fields}
}

func TestCertainJVMGenericSignatureOverflowFailsBeforeArtifacts(t *testing.T) {
	data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": wideSignatureSchema(240, 16)}}})
	if e != nil {
		t.Fatal(e)
	}
	rq := request(data, "Dto")
	rq.Options.Validation = true
	report, e := core.GenerateWithDiagnostics(rq)
	if e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Wide" {
		t.Fatal("certain JVM UTF8 signature overflow emitted unusable artifacts", e)
	}
}

func TestCertainJVMSignatureGuardPreservesCompiledResourceProfiles(t *testing.T) {
	for _, shape := range []struct{ width, depth int }{{200, 16}, {240, 12}} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": wideSignatureSchema(shape.width, shape.depth)}}})
			if e != nil {
				t.Fatal(e)
			}
			rq := request(data, "Dto")
			rq.Options.Validation = true
			rq.Options.ModelStyle = style
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("proven compiled profile rejected", shape, style, e)
			}
		}
	}
}

func TestCertainJVMSignatureOverflowRetainsNestedAndHTTPSourcePointers(t *testing.T) {
	wide := wideSignatureSchema(240, 16)
	for _, tc := range []struct {
		document map[string]any
		pointer  string
	}{
		{map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Outer": map[string]any{"properties": map[string]any{"cfg": wide}}}}}, "/components/schemas/Outer/properties/cfg"},
		{map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/wide": map[string]any{"post": map[string]any{"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": wide}}}, "responses": map[string]any{"204": map[string]any{}}}}}}, "/paths/~1wide/post/requestBody/content/application~1json/schema"},
		{map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/wide": map[string]any{"get": map[string]any{"responses": map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": wide}}}}}}}}, "/paths/~1wide/get/responses/200/content/application~1json/schema"},
	} {
		data, e := json.Marshal(tc.document)
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		report, e := core.GenerateWithDiagnostics(rq)
		if e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || e.(core.Diagnostics)[0].Pointer == nil || *e.(core.Diagnostics)[0].Pointer != tc.pointer {
			t.Fatal("certain signature overflow lost sourcePointer", tc.pointer, e)
		}
	}
}
