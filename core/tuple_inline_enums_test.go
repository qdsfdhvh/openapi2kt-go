package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleInlineEnumMembersAndTails(t *testing.T) {
	for _, leaf := range []struct{ schema, want string }{
		{`{"type":"string","enum":["a"]}`, "ProbeDtoInner"}, // Exact former rejection.
		{`{"enum":["ready","null"]}`, "ProbeDtoInner"},
		{`{"enum":[null,"ready"]}`, "ProbeDtoInner?"},
		{`{"type":["string","null"],"enum":[null,"ready"]}`, "ProbeDtoInner?"},
		{`{"type":"integer","enum":[1,2]}`, "kotlin.Int"}, // Exact former rejection.
		{`{"type":"integer","format":"int64","enum":[9007199254740993]}`, "kotlin.Long"},
		{`{"type":"number","format":"float","enum":[0.1,0.10000000149011612]}`, "kotlin.Float"},
		{`{"type":"number","enum":[0.1,-0.0]}`, "kotlin.Double"},
		{`{"type":"boolean","nullable":true,"enum":[true,false]}`, "kotlin.Boolean?"},
	} {
		for _, tc := range []struct{ prefix, tail, want string }{
			{`[` + leaf.schema + `]`, "", leaf.want},
			{`[` + leaf.schema + `]`, `,"items":false`, leaf.want},
			{`[` + leaf.schema + `]`, `,"items":true`, "kotlinx.serialization.json.JsonElement"},
			{`[` + leaf.schema + `,` + leaf.schema + `]`, "", "kotlinx.serialization.json.JsonElement"},
			{`[{"type":"boolean"},` + leaf.schema + `]`, `,"items":` + leaf.schema, leaf.want},
		} {
			for _, checked := range []bool{false, true} {
				for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
					rq := request(schemas(`{"Probe":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
					rq.Options.Validation, rq.Options.ModelStyle = checked, style
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(leaf, tc, checked, err)
					}
					if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
						t.Fatal(leaf, tc, source(t, got, "ProbeDto"))
					}
					again, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nondeterministic inline enum tuple", err)
					}
				}
			}
		}
	}
}

func TestTupleInlineEnumOwnDefaults(t *testing.T) {
	for _, leaf := range []struct{ schema, value string }{
		{`{"type":"string","enum":["ready"]}`, `["ready"]`},
		{`{"enum":[null,"ready"]}`, `[null,"ready"]`},
		{`{"type":"integer","enum":[1,2]}`, `[1,2]`},
		{`{"type":"number","format":"float","enum":[0.1]}`, `[0.1]`},
		{`{"type":"boolean","enum":[true,false]}`, `[false,true]`},
	} {
		for _, tail := range []string{"", `,"items":false`, `,"items":` + leaf.schema} {
			for _, checked := range []bool{false, true} {
				tuple := `{"type":"array","prefixItems":[` + leaf.schema + `]` + tail + `,"default":` + leaf.value + `}`
				rq := request(schemas(`{"Rows":`+tuple+`,"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")
				rq.Options.Validation = checked
				if _, err := core.Generate(rq); err != nil {
					t.Fatal(leaf, tail, checked, err)
				}
			}
		}
	}
	// The raw decision validates the original enum but does not constrain raw data.
	for _, tail := range []string{"", `,"items":true`, `,"items":false`} {
		tuple := `{"type":"array","prefixItems":[{"enum":["a"]},{"type":"boolean","enum":[true]}]` + tail + `,"default":[null,{"enum":"data"}]}`
		if _, err := core.Generate(request(schemas(`{"Rows":`+tuple+`,"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTupleInlineEnumDiscardedSourceGuards(t *testing.T) {
	for _, tc := range []struct{ leaf, pointer string }{
		{`{"enum":["a","a"]}`, "/enum/1"},
		{`{"enum":["a",1]}`, "/enum/1"},
		{`{"type":"string","enum":[]}`, "/enum"},
		{`{"type":"string","enum":[null,"a"]}`, "/enum/0"},
		{`{"type":"string","enum":["a"],"default":"a"}`, "/default"},
		{`{"type":"string","enum":["a"],"format":"email"}`, "/format"},
		{`{"type":"string","enum":["a"],"minLength":1}`, "/minLength"},
		{`{"type":"integer","enum":[1,"bad"]}`, "/enum/1"},
		{`{"type":"integer","enum":[2147483648]}`, "/enum/0"},
		{`{"type":"integer","enum":[1,1]}`, "/enum/1"},
		{`{"type":"integer","enum":[1],"minimum":1}`, "/minimum"},
		{`{"type":"boolean","enum":[true],"nullable":"bad"}`, "/nullable"},
	} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"string"}`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Probe":{"type":"array","prefixItems":[`+tc.leaf+`]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("discarded enum accepted", tc, tail, checked, err)
				}
				if d := err.(core.Diagnostics)[0]; d.Pointer == nil || *d.Pointer != "/components/schemas/Probe/prefixItems/0"+tc.pointer {
					t.Fatal(tc, tail, checked, err)
				}
			}
		}
	}
	for _, native := range []bool{false, true} {
		values := make([]string, 241)
		for i := range values {
			values[i] = fmt.Sprint(i)
			if !native {
				values[i] = fmt.Sprintf(`"v%d"`, i)
			}
		}
		kind := "string"
		if native {
			kind = "integer"
		}
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Probe":{"type":"array","prefixItems":[{"type":"`+kind+`","enum":[`+strings.Join(values, ",")+`]}],"items":true}}`), "Dto")
			rq.Options.Validation = checked
			if _, err := core.Generate(rq); err == nil {
				t.Fatal("raw projection hid vocabulary cap", native, checked)
			}
		}
	}
}

func TestTupleInlineEnumDefaultVocabularyAndSuppression(t *testing.T) {
	for _, tc := range []struct{ leaf, value string }{
		{`{"enum":["ready"]}`, `["other"]`},
		{`{"type":"integer","enum":[1,2]}`, `[3]`},
		{`{"type":"number","format":"float","enum":[0.1]}`, `[0.10000000149011612]`},
		{`{"type":"boolean","enum":[true]}`, `[false]`},
		{`{"enum":["ready"]}`, `[null]`},
	} {
		for _, required := range []string{"", `,"required":["rows"]`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[`+tc.leaf+`],"default":`+tc.value+`}}`+required+`}}`), "Dto")
				rq.Options.Validation = checked
				if _, err := core.Generate(rq); err == nil {
					t.Fatal("invalid enum annotation suppressed", tc, required, checked)
				}
			}
		}
	}
}

func TestTupleInlineEnumNativeNullVocabularyGate(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":["integer","null"],"enum":[null,1]}],"default":[null,1]}}`), "Dto")
		rq.Options.Validation = checked
		_, err := core.Generate(rq)
		if checked && err != nil || !checked && err == nil {
			t.Fatal("native null vocabulary lost existing profile gate", checked, err)
		}
	}
}

func TestTupleInlineEnumCumulativeAndConstructorBudgets(t *testing.T) {
	values := make([]string, 240)
	for i := range values {
		values[i] = fmt.Sprintf(`"v%d"`, i)
	}
	entries := make([]string, 64)
	for i := range entries {
		entries[i] = fmt.Sprintf(`"Rows%d":{"type":"array","prefixItems":[{"enum":[%s]}],"items":true}`, i, strings.Join(values, ","))
	}
	if _, err := core.Generate(request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")); err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" {
		t.Fatal("independent discarded vocabularies reset source budget", err)
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%d":{"type":"array","prefixItems":[{"type":"integer","enum":[1,2]}]}`, i)
		}
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
			rq.Options.Validation = checked
			_, err := core.Generate(rq)
			if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
				t.Fatal("constructor budget drift", width, checked, err)
			}
		}
	}
}
