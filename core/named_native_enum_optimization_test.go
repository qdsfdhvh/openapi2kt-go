package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNamedNativeEnumBorrowedChecksRetainReuseAndLockIdentity(t *testing.T) {
	components := `"CodeA":{"type":"integer","enum":[1,2,3,4],"minimum":2},"CodeB":{"type":"integer","enum":[1,2,3,4],"minimum":3},"Source":{"properties":{"a":{"$ref":"#/components/schemas/CodeA"},"b":{"$ref":"#/components/schemas/CodeB"}}}`
	r := request(schemas(`{`+components+`,"Alpha":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/a"}}},"Beta":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/a"}}},"Gamma":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/b"}}}}`), "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("checked named source identity erased")
	}
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/a"}}}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + components + `}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`)
	r = request(input, "Dto")
	r.Options.Validation = true
	r.Options.HoistSharedShapes = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
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
	r.Document = []byte(strings.Replace(string(input), `"minimum":2`, `"minimum":3`, 1))
	drift, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(drift, core.GenerationReport{}) {
		t.Fatal("canonical checks drift omitted", e)
	}
}
