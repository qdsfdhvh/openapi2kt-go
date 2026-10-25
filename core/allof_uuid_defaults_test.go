package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestAllOfUUIDOuterDefaults(t *testing.T) {
	input := schemas(`{"Record":{"allOf":[{"properties":{"id":{"type":"string","format":"uuid","pattern":"^123e4567-"}}},{"properties":{"id":{"type":"string","format":"uuid","maxLength":36}}}],"default":{"id":"123E4567-E89B-12D3-A456-426614174000"}}}`)
	rq := request(input, "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal("UUID outer allOf default rejected", e)
	}
}

const repeatedUUIDDefaults = `"allOf":[{"properties":{"id":{"type":"string","format":"uuid","minLength":36,"pattern":"^123e4567-","default":"123e4567-e89b-12d3-a456-426614174001"}}},{"properties":{"id":{"type":"string","format":"uuid","maxLength":36,"default":"123e4567-e89b-12d3-a456-426614174000"}}}]`

func TestAllOfUUIDOuterDefaultGuardsAndCanonicalChecks(t *testing.T) {
	for _, value := range []string{"123E4567-E89B-12D3-A456-426614174002", "123e4567e89b12d3a456426614174002"} {
		rq := request(schemas(`{"Factory":{`+repeatedUUIDDefaults+`,"default":{"id":"`+value+`"}}}`), "Dto")
		rq.Options.Validation = true
		if _, e := core.GenerateWithDiagnostics(rq); e != nil {
			t.Fatal("canonical UUID checks used source lexeme", value, e)
		}
	}
	for _, tc := range []struct{ shape, pointer string }{
		{`{"Factory":{` + repeatedUUIDDefaults + `,"default":{"id":"00000000-0000-0000-0000-000000000000"}}}`, "/components/schemas/Factory/default/id"},
		{`{"Factory":{` + repeatedUUIDDefaults + `,"default":{"id":"invalid"}}}`, "/components/schemas/Factory/default/id"},
		{`{"Factory":{` + strings.Replace(repeatedUUIDDefaults, "123e4567-e89b-12d3-a456-426614174001", "00000000-0000-0000-0000-000000000000", 1) + `,"default":{}}}`, "/components/schemas/Factory/allOf/0/properties/id/default"},
		{`{"Factory":{` + strings.Replace(repeatedUUIDDefaults, "123e4567-e89b-12d3-a456-426614174000", "00000000-0000-0000-0000-000000000000", 1) + `,"default":{}}}`, "/components/schemas/Factory/allOf/1/properties/id/default"},
	} {
		rq := request(schemas(tc.shape), "Dto")
		rq.Options.Validation = true
		report, e := core.GenerateWithDiagnostics(rq)
		if e == nil || len(report.Artifacts.Files) != 0 || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || e.(core.Diagnostics)[0].Pointer == nil || *e.(core.Diagnostics)[0].Pointer != tc.pointer {
			t.Fatal("UUID conjunction lost original default Pointer", tc.pointer, e)
		}
	}
	for _, fragment := range []string{
		strings.ReplaceAll(repeatedUUIDDefaults, `"format":"uuid"`, `"format":"date"`),
		strings.ReplaceAll(repeatedUUIDDefaults, `"format":"uuid"`, `"format":"uuid","enum":["123e4567-e89b-12d3-a456-426614174000"]`),
	} {
		rq := request(schemas(`{"Factory":{`+fragment+`,"default":{}}}`), "Dto")
		rq.Options.Validation = true
		if report, e := core.GenerateWithDiagnostics(rq); e == nil || len(report.Artifacts.Files) != 0 {
			t.Fatal("unrelated formatted/enum role widened", e)
		}
	}
	rq := request(schemas(`{"Factory":{`+repeatedUUIDDefaults+`,"default":{}}}`), "Dto")
	if report, e := core.GenerateWithDiagnostics(rq); e == nil || len(report.Artifacts.Files) != 0 {
		t.Fatal("UUID constraints bypassed opt-in", e)
	}
}
