package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleNullSourceBudgets(t *testing.T) {
	for _, hops := range []int{0, 2} {
		for _, count := range []int{45 - hops, 46 - hops} {
			for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
				parts := []string{`"Only":{"type":"null"}`}
				target := "Only"
				for i := 0; i < hops; i++ {
					parts = append(parts, fmt.Sprintf(`"Alias%d":{"$ref":"#/components/schemas/%s"}`, i, target))
					target = fmt.Sprintf("Alias%d", i)
				}
				leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]` + tail + `}`
				for i := 1; i < count; i++ {
					leaf = `{"type":"array","items":` + leaf + `}`
				}
				parts = append(parts, `"Rows":`+leaf, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`)
				for _, forward := range []bool{false, true} {
					if forward {
						parts[0], parts[len(parts)-1] = parts[len(parts)-1], parts[0]
					}
					_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
					if count == 45-hops && err != nil || count == 46-hops && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
						t.Fatal("null source erased cached original hops", hops, count, tail, forward, err)
					}
				}
			}
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[{"const":null}],"default":[null]}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("null tuple constructor cap", width, err)
		}
	}
	parts := []string{`"Only":{"type":"null"}`}
	for i := 0; i < 2000; i++ {
		parts = append(parts, fmt.Sprintf(`"R%04d":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Only"}],"items":true}`, i))
	}
	got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("null source cumulative capture reset", err)
	}
}

func TestTupleNullSources(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[{"type":"null"}]`, "", "List<kotlinx.serialization.json.JsonNull>"},
		{`[{"enum":[null]}]`, "", "List<kotlinx.serialization.json.JsonNull>"},
		{`[{"const":null}]`, "", "List<kotlinx.serialization.json.JsonNull>"},
		{`[{"type":"null"},{"enum":[null]},{"const":null}]`, "", "List<kotlinx.serialization.json.JsonNull>"},
		{`[{"type":"null","nullable":true}]`, "", "List<kotlinx.serialization.json.JsonNull?>"},
		{`[{"type":"null"},{"type":"string"}]`, "", "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"type":"null"}]`, `,"items":true`, "List<kotlinx.serialization.json.JsonElement>"},
		{`[{"type":"null"}]`, `,"items":{"type":"integer"}`, "List<kotlin.Int>"},
		{`[{"$ref":"#/components/schemas/Alias"}]`, "", "List<kotlinx.serialization.json.JsonNull>"},
		{`[{"$ref":"#/components/schemas/Only","nullable":true}]`, "", "List<kotlinx.serialization.json.JsonNull?>"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Only":{"type":"null"},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !strings.Contains(source(t, got, "RowsDto"), tc.want) {
				if err != nil {
					t.Fatal(tc, checked, err)
				}
				t.Fatal(tc, checked, source(t, got, "RowsDto"))
			}
			if len(got.Warnings) == 0 || got.Warnings[len(got.Warnings)-1].Code != "W_TUPLE_APPROXIMATION" {
				t.Fatal("lost null positional approximation", got.Warnings)
			}
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("null tuple source nondeterminism", e)
			}
		}
	}
}

func TestTupleNullOuterDefaults(t *testing.T) {
	for _, member := range []string{`{"type":"null"}`, `{"enum":[null]}`, `{"const":null}`, `{"$ref":"#/components/schemas/Only"}`} {
		for _, checked := range []bool{false, true} {
			for _, value := range []string{`[null,null]`, `[]`, `["null"]`, `[1]`} {
				input := schemas(`{"Only":{"type":"null"},"Rows":{"type":"array","prefixItems":[` + member + `],"default":` + value + `},"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[` + member + `],"default":` + value + `}},"required":["rows"]}}`)
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				valid := value == `[null,null]` || value == `[]`
				if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal(member, checked, value, err)
				}
			}
		}
	}
}

func TestTupleNullOriginalGrammarBeforeErasure(t *testing.T) {
	for _, tc := range []struct{ member, code, suffix string }{
		{`{"enum":[null,null]}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"type":"null","enum":[null,1]}`, "E_SCHEMA_INVALID", "/enum/1"},
		{`{"type":"null","default":null}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"const":null,"format":"uuid"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"type":"null","unknownWire":true}`, "E_SCHEMA_UNSUPPORTED", "/unknownWire"},
	} {
		input := schemas(`{"Rows":{"type":"array","prefixItems":[` + tc.member + `],"items":{"type":"integer"},"default":[]}}`)
		failure(t, input, tc.code, "/components/schemas/Rows/prefixItems/0"+tc.suffix)
	}
}
