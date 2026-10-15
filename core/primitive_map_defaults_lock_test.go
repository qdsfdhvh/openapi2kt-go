package core_test

import (
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestPrimitiveMapDefaultsMappedLockReplayAndChangedValue(t *testing.T) {
	for _, checked := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchecked", true: "checked"}[checked], func(t *testing.T) {
			testPrimitiveMapDefaultsLock(t, checked)
		})
	}
}

func testPrimitiveMapDefaultsLock(t *testing.T, checked bool) {
	t.Helper()
	op := func(id, value string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"object","additionalProperties":{"type":"number","format":"float"},"default":` + value + `}}}}}}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op("alpha", `{"b":1.25,"a":1}`) + `,"/b":` + op("beta", `{"b":1.25,"a":1}`) + `}}`)
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
	r.Document = []byte(strings.ReplaceAll(string(doc), `{"b":1.25,"a":1}`, `{"a":1.000000001,"b":1.250}`))
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("Equivalent mapped map default changed owned lock", err)
	}
	if checked {
		r.Document = []byte(strings.ReplaceAll(string(doc), `"minimum":1`, `"minimum":0`))
		changed, err := core.GenerateWithDiagnostics(r)
		if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
			t.Fatal("Changed map value checks reused a frozen shape", err)
		}
	}
	r.Document = []byte(strings.ReplaceAll(string(doc), `{"b":1.25,"a":1}`, `{"a":2,"b":1.25}`))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("Map value changed frozen shape silently", err)
	}
}
