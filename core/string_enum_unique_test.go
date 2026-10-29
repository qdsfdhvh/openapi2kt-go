package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func stringEnumUniqueGraduation(t *testing.T, input []byte) {
	t.Helper()
	rq := request(input, "Dto")
	if _, err := core.GenerateWithDiagnostics(rq); err == nil {
		t.Fatal("missing uniqueness opt-in")
	}
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("original direct enum uniqueness did not graduate", err)
	}
}

func TestDirectStringEnumUniqueSourceExclusions(t *testing.T) {
	for _, leaf := range []string{
		`{"type":"string","enum":["ready"],"format":"date"}`,
		`{"type":"string","enum":["ready"],"x-native-type":"bigint"}`,
		`{"type":"string","const":"ready"}`,
		`{"anyOf":[{"enum":["ready"]},{"type":"string"}]}`,
	} {
		rq := request(schemas(`{"Code":{"enum":["ready"]},"Record":{"properties":{"value":{"type":"array","items":`+leaf+`,"uniqueItems":true}}}}`), "Dto")
		rq.Options.Validation = true
		if metadataStringEnumFixture(leaf) {
			metadataStringEnumGraduation(t, rq)
			continue
		}
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("projected or referenced leaf acquired enum uniqueness", leaf, r, err)
		}
	}
	aliasFailure(t, schemas(`{"Code":{"type":"string","x-kotlin-value-class":true},"Record":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Code"},"uniqueItems":true}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/uniqueItems")
}

func TestDirectStringEnumUniqueNoDropAndBudgets(t *testing.T) {
	shape := `{"type":"array","items":{"enum":["ready"]},"uniqueItems":true}`
	aliasFailure(t, schemas(`{"Record":{"properties":{"v":{"type":"string"}},"additionalProperties":`+shape+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/additionalProperties/uniqueItems")
	aliasFailure(t, schemas(`{"Record":{"allOf":[{"properties":{"v":`+shape+`}}],"properties":{"v":{"type":"string"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/allOf/0/properties/v/uniqueItems")
	for _, edges := range []int{16, 17} {
		field := shape
		for i := 1; i < edges; i++ {
			field = `{"type":"array","items":` + field + `}`
		}
		rq := request(fieldDoc(field), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (edges == 16) || err != nil && (len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
			t.Fatal("enum unique checked edge budget lost", edges, r, err)
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":%s`, i, shape)
		}
		rq := request(schemas(`{"Record":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (width == 240) || err != nil && (len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("enum unique slot budget lost", width, r, err)
		}
	}
}

func TestDirectStringEnumUniqueSources(t *testing.T) {
	stringEnumUniqueGraduation(t, fieldDoc(`{"type":"array","items":{"enum":["one"]},"uniqueItems":true,"default":[]}`))
	stringEnumUniqueGraduation(t, fieldDoc(`{"type":"array","items":{"type":"array","items":{"enum":["one"]},"uniqueItems":true},"default":[]}`))
	for _, leaf := range []string{
		`{"type":"string","enum":["ready","done"]}`,
		`{"enum":["ready",null],"nullable":true}`,
		`{"type":"string","enum":["ready","x"],"minLength":2}`,
	} {
		rq := request(fieldDoc(`{"type":"array","items":`+leaf+`,"uniqueItems":true}`), "Dto")
		rq.Options.Validation = true
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal("direct String enum uniqueness should generate", leaf, err)
		}
		text := source(t, r, "RecordDto")
		if !strings.Contains(text, "serialName") || !strings.Contains(text, "`openapi-kt uniqueItems`(value.map") {
			t.Fatal("enum payload comparison lost", text)
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(r, again) {
			t.Fatal("enum uniqueness output drift", err)
		}
		rq.Options.Validation = false
		if r, err := core.GenerateWithDiagnostics(rq); err == nil || len(r.Artifacts.Files) != 0 {
			t.Fatal("validation opt-in lost", r, err)
		}
	}
}

func TestDirectStringEnumUniqueDefaultMembershipAndDuplicates(t *testing.T) {
	for _, values := range []string{`["ready","ready"]`, `["ready","\u0072eady"]`, `[null,null]`, `["future"]`} {
		for _, nested := range []bool{false, true} {
			leaf := `{"enum":["ready",null],"nullable":true}`
			field := `{"type":"array","items":` + leaf + `,"uniqueItems":true,"default":` + values + `}`
			if nested {
				field = `{"type":"array","items":{"type":"array","items":` + leaf + `,"uniqueItems":true},"default":[` + values + `]}`
			}
			rq := request(fieldDoc(field), "Dto")
			rq.Options.Validation = true
			r, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(r.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || !strings.Contains(*err.(core.Diagnostics)[0].Pointer, "/default/") {
				t.Fatal("enum duplicate/membership default lost", values, nested, r, err)
			}
		}
	}
}
