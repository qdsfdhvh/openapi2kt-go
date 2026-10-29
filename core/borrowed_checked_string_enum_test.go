package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCheckedStringEnumLeavesAndCollections(t *testing.T) {
	for _, leaf := range []string{
		`{"enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}`,
		`{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"}`,
	} {
		for _, owner := range []string{"AOwner", "ZOwner"} {
			body := `{"` + owner + `":{"properties":{"word":LEAF,"tree":{"type":"array","items":{"additionalProperties":LEAF}}}},"Borrowed":{"$ref":"#/components/schemas/OWNER/properties/tree"},"Word":{"$ref":"#/components/schemas/OWNER/properties/word"},"Use":{"properties":{"word":{"$ref":"#/components/schemas/OWNER/properties/word"},"tree":{"$ref":"#/components/schemas/Borrowed"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Word"}}}}}`
			body = strings.ReplaceAll(strings.ReplaceAll(body, "LEAF", leaf), "OWNER", owner)
			rq := request(schemas(body), "Dto")
			rq.Options.Validation = true
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("actual checked lexical borrowing refused", owner, err)
			}
			text := source(t, got, "UseDto")
			if !strings.Contains(text, owner+"Dto.Word") || !strings.Contains(text, "serialName.let") {
				t.Fatal("checked borrowing lost actual lexical classifier/checks", text)
			}
			if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("checked enum borrowing nondeterminism", err)
			}
			rq.Options.Validation = false
			if got, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("checked borrowing bypassed opt-in", err)
			}
		}
	}
}

func TestBorrowedCheckedStringEnumFormerGuards(t *testing.T) {
	field := `{"type":"array","items":{"enum":["one","x"],"minLength":2},"default":["one"]}`
	for _, body := range []string{
		`{"Holder":{"properties":{"codes":` + field + `}},"Use":{"properties":{"value":{"$ref":"#/components/schemas/Holder/properties/codes/items"}}}}`,
		`{"Holder":{"properties":{"codes":` + field + `}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Holder/properties/codes/items"}}}`,
		`{"Owner":{"properties":{"state":{"enum":["one"],"minLength":2}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/state"}}`,
		`{"Use":{"properties":{"leaf":{"enum":["one"],"minLength":2},"value":{"type":"array","items":{"type":"array","items":{"$ref":"#/components/schemas/Use/properties/leaf"}}}}}}`,
		`{"Value":{"properties":{"state":{"enum":["one"],"minLength":2},"bad":{"$ref":"#/components/schemas/Value/properties/state"}},"default":{"state":"one"}}}`,
	} {
		rq := request(schemas(body), "Dto")
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("original checked-source reference refusal did not graduate", body, err)
		}
	}
}

func TestBorrowedCheckedStringEnumIndependentDefaults(t *testing.T) {
	body := `{"Owner":{"properties":{"word":{"type":"string","format":"date","enum":["one","two","x"],"minLength":2,"maxLength":6,"default":"one"},"tree":{"type":"array","items":{"additionalProperties":{"enum":["one","two","x"],"minLength":2}}}}},"Word":{"$ref":"#/components/schemas/Owner/properties/word","default":"two"},"Maybe":{"$ref":"#/components/schemas/Word","nullable":true,"default":null},"Tree":{"$ref":"#/components/schemas/Owner/properties/tree"},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Word","default":"one"},"rows":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["two"]},"tree":{"$ref":"#/components/schemas/Tree","default":[{"field":"one"}]}},"default":{"word":"two","rows":["one"],"tree":[{"outer":"two"}]}}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("checked original source/factory/consumer defaults rejected", err, *err.(core.Diagnostics)[0].Pointer)
	}
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default":"one"`, `"default":"x"`, "/components/schemas/Owner/properties/word/default"},
		{`"default":"two"`, `"default":"future"`, "/components/schemas/Word/default"},
		{`"field":"one"`, `"field":"x"`, "/components/schemas/Use/properties/tree/default/0/field"},
		{`"outer":"two"`, `"outer":"x"`, "/components/schemas/Use/default/tree/0/outer"},
	} {
		enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
}
