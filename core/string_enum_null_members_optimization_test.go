package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumNullMembersKeepPublicVocabularyAndLockIdentity(t *testing.T) {
	r := request(schemas(`{"Alpha":{"properties":{"state":{"type":["string","null"],"enum":["ready",null,"done"]}}},"Beta":{"properties":{"state":{"type":["string","null"],"enum":["ready","done"]}}},"Gamma":{"properties":{"state":{"type":["string","null"],"enum":["ready",null,"other"]}}}}`), "Dto")
	r.Options.MergeIdenticalModels = true
	got, e := core.GenerateWithDiagnostics(r)
	if e != nil || !strings.Contains(source(t, got, "BetaDto"), "typealias BetaDto") || strings.Contains(source(t, got, "GammaDto"), "typealias GammaDto") {
		t.Fatal(e)
	}
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":["string","null"],"enum":["ready","done"]}},"required":["value"]}}}}}}}`
	}
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`)
	r = request(input, "Dto")
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
	r.Document = []byte(strings.ReplaceAll(string(input), `"enum":["ready","done"]`, `"enum":["ready",null,"done"]`))
	replayed, e := core.GenerateWithDiagnostics(r)
	if e != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatal("Codec null intent changed public vocabulary identity", e)
	}
	r.Document = []byte(strings.ReplaceAll(string(r.Document), `"enum":["ready",null,"done"]`, `"enum":["ready",null,"other"]`))
	changed, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("String vocabulary drift reused name lock", e)
	}
}
