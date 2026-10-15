package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNativeEnumMembershipLockReplayVocabularyDrift(t *testing.T) {
	operation := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"code":{"type":"integer","enum":[1,2]}}}}}}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + operation("alpha") + `,"/b":` + operation("beta") + `}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	r.Options.MergeIdenticalModels = true
	r.Options.DeduplicateModels = true
	first, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.PreviousNameLock = first.Artifacts.NameLock
	replay, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatal(err)
	}
	r.Document = []byte(strings.ReplaceAll(string(doc), `"enum":[1,2]`, `"enum":[1,3]`))
	changed, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("enum vocabulary was omitted from lock identity", err)
	}
}

func TestNativeEnumCheckedLiteralAndOriginalSourceGuards(t *testing.T) {
	for _, schema := range []string{
		`{"type":"number","enum":[1e513]}`,
		`{"type":"number","enum":[1e-513]}`,
		`{"type":"number","format":"float","enum":[1e39]}`,
		`{"type":"integer","enum":[null]}`,
		`{"type":"integer","enum":[1,1]}`,
		`{"type":"integer","enum":[1],"default":2}`,
	} {
		r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"value":`+schema+`}}}}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(schema, err)
		}
	}
	// Preserve the historical valid enum/minimum input as explicitly graduated
	// coverage: both the vocabulary and retained bound must be emitted.
	graduated := request(fieldDoc(`{"type":"integer","enum":[1],"minimum":0}`), "Dto")
	graduated.Options.Validation = true
	accepted, e := core.GenerateWithDiagnostics(graduated)
	if e != nil || !strings.Contains(source(t, accepted, "RecordDto"), "value == 1") || !strings.Contains(source(t, accepted, "RecordDto"), "value >= 0") {
		t.Fatal(accepted, e)
	}
	// Checked enum discarded by allOf must not silently disappear.
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"allOf":[{"properties":{"value":{"type":"integer","enum":[1,2]}}},{"properties":{"value":{"type":"integer"}}}]}}}}`)
	r := request(doc, "Dto")
	r.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(r)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("discarded enum enforcement dropped", err)
	}
}
