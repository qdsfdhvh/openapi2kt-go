package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestPrimitiveArrayDefaultsMappedLockReplayAndChangedSequence(t *testing.T) {
	for _, checked := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchecked", true: "checked"}[checked], func(t *testing.T) {
			testPrimitiveArrayDefaultsLock(t, checked)
		})
	}
}

func testPrimitiveArrayDefaultsLock(t *testing.T, checked bool) {
	t.Helper()
	op := func(id, value string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"array","items":{"type":"number","format":"float"},"default":` + value + `}}}}}}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op("alpha", "[1.25,1]") + `,"/b":` + op("beta", "[1.25,1]") + `}}`)
	if checked {
		doc = []byte(strings.ReplaceAll(string(doc), `"format":"float"`, `"format":"float","minimum":1`))
	}
	r := request(doc, "Dto")
	r.Options.Validation = checked
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	r.Options.DeduplicateModels = true
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal(got, err)
	}
	r.PreviousNameLock = got.Artifacts.NameLock
	r.Document = []byte(strings.ReplaceAll(string(doc), "[1.25,1]", "[1.250,1.000000001]"))
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("Equivalent mapped array default changed owned lock", err)
	}
	if checked {
		r.Document = []byte(strings.ReplaceAll(string(doc), `"minimum":1`, `"minimum":0`))
		changed, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
			t.Fatal("Changed array element checks reused a frozen shape", err)
		}
	}
	r.Document = []byte(strings.ReplaceAll(string(doc), "[1.25,1]", "[1,1.25]"))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("Array order changed frozen shape silently", err)
	}
}
