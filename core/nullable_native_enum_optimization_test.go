package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableNativeEnumsRetainReuseAndLockedNullViews(t *testing.T) {
	components := `"Code":{"type":["integer","null"],"enum":[1,2]},"False":{"$ref":"#/components/schemas/Code","nullable":false}`
	r := request(schemas(`{`+components+`,"Alpha":{"properties":{"value":{"$ref":"#/components/schemas/Code"}},"required":["value"]},"Beta":{"properties":{"value":{"$ref":"#/components/schemas/Code"}},"required":["value"]},"Gamma":{"properties":{"value":{"$ref":"#/components/schemas/False"}},"required":["value"]}}`), "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("nullable views merged")
	}
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/False"}},"required":["value"]}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + components + `}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`)
	r = request(input, "Dto")
	r.Options.Validation = true
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	first, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	r.PreviousNameLock = first.Artifacts.NameLock
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(first, again) {
		t.Fatal(e)
	}
	r.Document = []byte(strings.Replace(string(input), `"nullable":false`, `"nullable":true`, 1))
	drift, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(drift, core.GenerationReport{}) {
		t.Fatal("nullable shape drift absent", e)
	}
}
