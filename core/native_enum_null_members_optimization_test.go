package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumNullMemberChecksKeepReuseAndLockIdentity(t *testing.T) {
	r := request(schemas(`{"Alpha":{"properties":{"value":{"type":["integer","null"],"enum":[1,null,2]}}},"Beta":{"properties":{"value":{"type":["integer","null"],"enum":[1,2]}}},"Gamma":{"properties":{"value":{"type":["integer","null"],"enum":[1,null,3]}}}}`), "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil || !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("Mapped non-null membership identity changed", e)
	}
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"$ref":"#/components/schemas/Code"}},"required":["value"]}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":["integer","null"],"enum":[1,2]}}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`)
	r = request(input, "Dto")
	r.Options.Validation = true
	r.Options.HoistSharedShapes = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
	r.Options.LockHoistedNames = true
	r.Options.QualifyHoistedNames = true
	first, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	r.PreviousNameLock = first.Artifacts.NameLock
	r.Document = []byte(strings.Replace(string(input), `"enum":[1,2]`, `"enum":[1,null,2]`, 1))
	again, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("Redundant null member changed nullable public checks or lock identity", e)
	}
	r.Document = []byte(strings.Replace(string(r.Document), `"enum":[1,null,2]`, `"enum":[1,null,3]`, 1))
	drift, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(drift, core.GenerationReport{}) {
		t.Fatal("Non-null membership drift replayed checked name lock", e)
	}
}
