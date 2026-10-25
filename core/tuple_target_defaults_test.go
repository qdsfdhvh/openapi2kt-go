package core_test

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleTargetOwnDefaults(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/tuple-target-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.ModelStyle = style
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(err, *err.(core.Diagnostics)[0].Pointer)
			}
			for _, name := range []string{"RowsDto", "PlainRowsDto", "TextRowsDto", "TimeRowsDto", "EnumRowsDto", "AllFactoryRowsDto"} {
				if !strings.Contains(source(t, got, name), "List<") {
					t.Fatal("lost target actual type", name)
				}
			}
			if !strings.Contains(source(t, got, "PlainDto"), "fun default()") || !strings.Contains(source(t, got, "AllFactoryDto"), "fun default()") {
				t.Fatal("lost own component factory")
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("target default nondeterminism", err)
			}
		}
	}
}

func TestTupleTargetDefaultSourceAndOwnRoleBoundary(t *testing.T) {
	for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
		input := schemas(`{"Item":{"properties":{"value":{"type":"string"}},"required":["value"],"default":{}},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]` + tail + `}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Item/default")
		input = schemas(`{"Item":{"properties":{"value":{"type":"string"}},"default":{}},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]` + tail + `}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("exact original valid own-target negative did not graduate", err)
		}
		// A positional source is not the target component: it cannot borrow the
		// target's annotation permission for its own newly written default.
		bad := bytes.Replace(input, []byte(`"$ref":"#/components/schemas/Item"`), []byte(`"$ref":"#/components/schemas/Item","default":{}`), 1)
		failure(t, bad, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Probe/prefixItems/0/default")
	}
}

func TestTupleTargetGraduatedOriginalDefaults(t *testing.T) {
	for _, tc := range []struct{ name, body string }{{"Item", `{"properties":{"value":{"type":"string"}},"default":{}}`}, {"Choice", `{"type":"string","enum":["a"],"default":"a"}`}, {"Text", `{"type":"string","default":"hidden"}`}} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"`+tc.name+`":`+tc.body+`,"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/`+tc.name+`"}]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				if _, err := core.Generate(rq); err != nil {
					t.Fatal("exact original target-default negative did not graduate", tc, tail, checked, err)
				}
			}
		}
	}
}

func TestTupleTargetDefaultStrictLiteralAndFactoryBudgets(t *testing.T) {
	for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
		input := schemas(`{"Choice":{"type":"string","enum":["a"],"default":"future"},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Choice"}]` + tail + `}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Choice/default")
		input = schemas(`{"Text":{"type":"string","default":"` + strings.Repeat("x", 32769) + `"},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Text"}]` + tail + `}}`)
		failure(t, input, "E_MODEL_LIMIT", "/components/schemas/Text/default")
	}
}

func TestBorrowedEnumOwnFactoryKeepsOriginalVocabulary(t *testing.T) {
	for _, items := range []string{`"items":{"$ref":"#/components/schemas/Choice"}`, `"prefixItems":[{"$ref":"#/components/schemas/Choice"}]`} {
		input := schemas(`{"Choice":{"type":"string","enum":["a"],"default":"a"},"Rows":{"type":"array",` + items + `}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("original owned factory lost borrowed enum source authority", err)
		}
	}
}
