package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumBoundsRetainDistinctReuseAndLockedIdentity(t *testing.T) {
	doc := schemas(`{"CodeA":{"type":"integer","enum":[1,2,3,4],"minimum":2},"CodeB":{"type":"integer","enum":[1,2,3,4],"minimum":3},"Alpha":{"properties":{"value":{"$ref":"#/components/schemas/CodeA"}}},"Beta":{"properties":{"value":{"$ref":"#/components/schemas/CodeA"}}},"Gamma":{"properties":{"value":{"$ref":"#/components/schemas/CodeB"}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal("bounds identity erased")
	}
	op := func(min string) []byte {
		operation := func(id string) string {
			return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"integer","enum":[1,2,3,4],"minimum":` + min + `}}}}}}}}}`
		}
		return []byte(`{"openapi":"3.1.0","paths":{"/a":` + operation("alpha") + `,"/b":` + operation("beta") + `}}`)
	}
	r = request(op("2"), "Dto")
	r.Options.Validation = true
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	locked, e := core.GenerateWithDiagnostics(r)
	if e != nil {
		t.Fatal(e)
	}
	r.PreviousNameLock = locked.Artifacts.NameLock
	replay, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(locked, replay) {
		t.Fatal("lock replay", e)
	}
	r.Document = op("3")
	if _, e := core.GenerateWithDiagnostics(r); e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
		t.Fatal("locked bound drift", e)
	}
}
