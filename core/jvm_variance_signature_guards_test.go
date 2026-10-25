package core_test

import (
	"encoding/json"
	"openapi-kt/core"
	"testing"
)

func TestJVMNestedCollectionVarianceSignatureOverflow(t *testing.T) {
	data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": wideSignatureSchema(204, 16)}}})
	if e != nil {
		t.Fatal(e)
	}
	rq := request(data, "Dto")
	rq.Options.Validation = true
	report, e := core.GenerateWithDiagnostics(rq)
	if e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
		t.Fatal("204xList16 constructor overflow still emitted", e)
	}
}

func TestJVMNestedListVariancePreservesBorderProfiles(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Wide": wideSignatureSchema(203, 16)}}})
		if e != nil {
			t.Fatal(e)
		}
		rq := request(data, "Dto")
		rq.Options.Validation = true
		rq.Options.ModelStyle = style
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("feasible203x16 rejected", style, e)
		}
	}
}
