package core_test

import (
	"openapi-kt/core"
	"testing"
)

func TestAllOfDefaultOverridesOriginalTypedAndOuterControls(t *testing.T) {
	for _, own := range []bool{false, true} {
		annotation := ""
		if own {
			annotation = `,"default":{"id":"a"}`
		}
		item := `{"allOf":[{"properties":{"id":{"type":"string"},"version":{"type":"integer","default":0}}},{"properties":{"version":{"type":"integer","minimum":0,"default":1}}}],"required":["id"]` + annotation + `}`
		rq := request(schemas(`{"Item":`+item+`}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Errorf("original homogeneous override constructor vs outer annotation=%v: %v", own, e)
		}
	}
}

const allOfOverrideDuplicateA = `{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"a":{"type":"string"}}}]}`
const allOfOverrideDuplicateName = `{"allOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"name":{"type":"string"}}}]}`
const allOfOverrideDuplicateLabel = `{"allOf":[{"properties":{"label":{"type":"string"}}},{"properties":{"label":{"type":"string"}}}]}`

func allOfDefaultOverridePositive(t *testing.T, rq core.GenerationRequest) {
	t.Helper()
	report, e := core.GenerateWithDiagnostics(rq)
	if e != nil || len(report.Artifacts.Files) == 0 {
		t.Fatal("exact original homogeneous allOf default role", e)
	}
	found := false
	for _, w := range report.Warnings {
		found = found || w.Code == "W_ALL_OF_APPROXIMATION"
	}
	if !found {
		t.Fatal("lost original allOf approximation warning")
	}
}

func TestAllOfDefaultOverridesStrictOriginalAnnotationsAndWinningChecks(t *testing.T) {
	for _, tc := range []struct{ first, last, outer, code, pointer string }{
		{`{"type":"integer","default":"bad"}`, `{"type":"integer","minimum":1,"default":2}`, `{}`, "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/0/properties/value/default"},
		{`{"type":"integer","default":0}`, `{"type":"integer","minimum":1,"default":0}`, `{}`, "E_SCHEMA_INVALID", "/components/schemas/Record/allOf/1/properties/value/default"},
		{`{"type":"integer","default":0}`, `{"type":"integer","minimum":1,"default":2}`, `{"value":0}`, "E_SCHEMA_INVALID", "/components/schemas/Record/default/value"},
		{`{"type":"string","default":"early"}`, `{"type":"string","minLength":2,"default":"ok"}`, `{"value":"x"}`, "E_SCHEMA_INVALID", "/components/schemas/Record/default/value"},
	} {
		rq := request(schemas(`{"Record":{"allOf":[{"properties":{"value":`+tc.first+`}},{"properties":{"value":`+tc.last+`}}],"default":`+tc.outer+`}}`), "Dto")
		rq.Options.Validation = true
		r, e := core.GenerateWithDiagnostics(rq)
		ds, ok := e.(core.Diagnostics)
		if !ok || len(ds) != 1 || ds[0].Code != tc.code || ds[0].Pointer == nil || *ds[0].Pointer != tc.pointer || len(r.Artifacts.Files) != 0 {
			t.Fatal("strict original/winning override annotation", tc, e)
		}
	}
}
