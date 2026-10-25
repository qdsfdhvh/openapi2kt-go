package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestJVMDataCopyReturnNameOverflow(t *testing.T) {
	data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": wideSignatureSchema(203, 16)}}})
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(data, "Dto")
		rq.Options.Validation = true
		rq.Options.ModelStyle = style
		rq.Options.PackageName = "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
		report, e := core.GenerateWithDiagnostics(rq)
		if style == core.DataClass && (e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("unrepresentable Data copy signature emitted", e)
		}
		if style == core.PlainClass && e != nil {
			t.Fatal("Plain has no Data copy API", e)
		}
	}
}

func TestDataCopyPackageGuardUsesFinalStyleAndOriginalPointer(t *testing.T) {
	packageName := "org.example." + strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 60)+".", 5), ".")
	wide := wideSignatureSchema(203, 16)
	for _, tc := range []struct {
		document map[string]any
		pointer  string
	}{
		{map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide/~": wide}}}, "/components/schemas/Wide~1~0"},
		{map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Outer": map[string]any{"properties": map[string]any{"cfg": wide}}}}}, "/components/schemas/Outer/properties/cfg"},
		{map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/wide": map[string]any{"post": map[string]any{"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": wide}}}, "responses": map[string]any{"204": map[string]any{}}}}}}, "/paths/~1wide/post/requestBody/content/application~1json/schema"},
	} {
		data, e := json.Marshal(tc.document)
		if e != nil {
			t.Fatal(e)
		}
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for mask := 0; mask < 8; mask++ {
				rq := request(data, "Dto")
				rq.Options.PackageName = packageName
				rq.Options.Validation = true
				rq.Options.ModelStyle = style
				rq.Options.DeduplicateModels = mask&1 != 0
				rq.Options.HoistSharedShapes = mask&2 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				rq.Options.MergeIdenticalModels = mask&4 != 0
				report, e := core.GenerateWithDiagnostics(rq)
				if style == core.DataClass {
					if e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || e.(core.Diagnostics)[0].Pointer == nil || *e.(core.Diagnostics)[0].Pointer != tc.pointer {
						t.Fatal("retained Data copy lost final style/source", mask, tc.pointer, e)
					}
				} else {
					if e != nil {
						t.Fatal("Plain constructor rejected by Data copy proof", mask, e)
					}
					again, e := core.GenerateWithDiagnostics(rq)
					if e != nil || !reflect.DeepEqual(report, again) {
						t.Fatal("diagnostic metadata changed determinism", e)
					}
				}
			}
		}
	}
}
