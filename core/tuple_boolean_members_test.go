package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleBooleanPositionalMembers(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[true]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[true,true]`, `,"items":false`, "kotlinx.serialization.json.JsonElement"},
		{`[false]`, "", "ProbeDtoInner"},
		{`[false,false]`, `,"items":false`, "ProbeDtoInner"},
		{`[false]`, `,"items":true`, "kotlinx.serialization.json.JsonElement"},
		{`[false,true]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[false,{"type":"string"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[false,true]`, `,"items":{"type":"integer"}`, "kotlin.Int"},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Probe":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
				rq.Options.ModelStyle = style
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(tc, checked, err)
				}
				if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
					t.Fatal(tc, source(t, got, "ProbeDto"))
				}
				if tc.want == "ProbeDtoInner" && !strings.Contains(source(t, got, "ProbeDtoInner"), "admits no JSON value") {
					t.Fatal("false element lost zero-domain codec")
				}
				if strings.Contains(tc.prefix, "false") && tc.want != "ProbeDtoInner" {
					warned := false
					for _, w := range got.Warnings {
						warned = warned || w.Code == "W_TUPLE_APPROXIMATION" && strings.Contains(w.Message, "Boolean false prefix restrictions")
					}
					if !warned {
						t.Fatal("projected false positions lost their explicit warning", got.Warnings)
					}
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Boolean member nondeterminism", e)
				}
			}
		}
	}
}

func TestTupleBooleanNullableAliasFactories(t *testing.T) {
	input := schemas(`{"Rows":{"type":"array","prefixItems":[false],"nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Rows","nullable":false,"default":[]},"Outer":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null}}`)
	if _, err := core.Generate(request(input, "Dto")); err != nil {
		t.Fatal("Boolean tuple nullable factory view", err)
	}
	input = []byte(strings.Replace(string(input), `"nullable":false,"default":[]`, `"nullable":false,"default":null`, 1))
	failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Narrow/default")
}

func TestTupleBooleanOriginalSourceAndCountGuards(t *testing.T) {
	for _, prefix := range []string{`[true,{"type":"string","unknownTuple":true}]`, `[false,{"type":"string","unknownTuple":true}]`} {
		input := schemas(`{"Rows":{"type":"array","prefixItems":` + prefix + `,"items":true}}`)
		failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/prefixItems/1/unknownTuple")
	}
	for _, root := range []bool{false, true} {
		tuple := `{"type":"array","prefixItems":[false],"minItems":1,"default":[]}`
		declaration, pointer := `"Rows":`+tuple, "/components/schemas/Rows/default"
		if !root {
			declaration = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			pointer = "/components/schemas/Holder/properties/rows/default"
		}
		rq := request(schemas(`{`+declaration+`}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.Generate(rq); err == nil || *err.(core.Diagnostics)[0].Pointer != pointer {
			t.Fatal("false empty default skipped count checks", root, err)
		}
	}
	for _, prefix := range []string{`[true]`, `[false]`, `[true,false]`} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":`+prefix+`,"uniqueItems":true}}`), "Dto")
		rq.Options.Validation = true
		if _, err := core.Generate(rq); err == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/uniqueItems" {
			t.Fatal("Boolean member created positional uniqueness authority", prefix, err)
		}
		rq.Document = []byte(strings.Replace(string(rq.Document), "3.1.0", "3.0.3", 1))
		if _, err := core.Generate(rq); err == nil {
			t.Fatal("3.0 Boolean tuple admitted")
		}
	}
}

func TestTupleBooleanConstructorAndCaptureBudgets(t *testing.T) {
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":{"type":"array","prefixItems":[false],"default":[]}`, i)
		}
		rq := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
		_, err := core.Generate(rq)
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("false tuple constructor cap", width, err)
		}
	}
	prefix := strings.TrimSuffix(strings.Repeat("false,", 9000), ",")
	if _, err := core.Generate(request(schemas(`{"Rows":{"type":"array","prefixItems":[`+prefix+`],"items":true}}`), "Dto")); err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
		t.Fatal("discarded Boolean source capture cap", err)
	}
}

func TestTupleBooleanMemberOwnDefaults(t *testing.T) {
	for _, tc := range []struct{ prefix, value string }{
		{`[false]`, `[]`},
		{`[true]`, `[null,true,{"default":"data"}]`},
		{`[false,true]`, `[null,false]`},
	} {
		tuple := `{"type":"array","prefixItems":` + tc.prefix + `,"items":false,"default":` + tc.value + `}`
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Rows":`+tuple+`,"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")
			rq.Options.Validation = checked
			if _, err := core.Generate(rq); err != nil {
				t.Fatal(tc, checked, err)
			}
		}
	}
	for _, value := range []string{`[null]`, `[false]`, `[{}]`, `[[]]`, `[1]`, `["false"]`} {
		for _, required := range []string{"", `,"required":["rows"]`} {
			input := schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[false],"default":` + value + `}}` + required + `}}`)
			failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/rows/default/0")
		}
	}
}
