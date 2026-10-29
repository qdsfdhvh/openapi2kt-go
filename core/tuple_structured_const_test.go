package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleStructuredConstReferenceNullViews(t *testing.T) {
	for _, override := range []bool{false, true} {
		for _, value := range []string{`[{"fixed":1}]`, `[null]`, `[{"fixed":2}]`} {
			input := schemas(fmt.Sprintf(`{"Only":{"const":{"fixed":1},"nullable":true},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias","nullable":%v}],"default":%s}}`, override, value))
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			valid := value == `[{"fixed":1}]` || override && value == `[null]`
			if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(override, value, err)
			}
		}
	}
}

func TestTupleStructuredConstOriginalGuards(t *testing.T) {
	for _, tc := range []struct{ member, code, suffix string }{
		{`{"type":"string","const":{"fixed":1}}`, "E_SCHEMA_INVALID", "/const"},
		{`{"const":{"fixed":1},"format":"uuid"}`, "E_SCHEMA_UNSUPPORTED", "/format"},
		{`{"const":[1],"default":[1]}`, "E_SCHEMA_UNSUPPORTED", "/default"},
		{`{"const":[1],"unknownWire":true}`, "E_SCHEMA_UNSUPPORTED", "/unknownWire"},
	} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			failure(t, schemas(`{"Rows":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`,"default":[]}}`), tc.code, "/components/schemas/Rows/prefixItems/0"+tc.suffix)
		}
	}
}

func TestTupleStructuredConstDepthAndWidth(t *testing.T) {
	for _, count := range []int{45, 46} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Only"}]` + tail + `}`
			for i := 1; i < count; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			for _, forward := range []bool{false, true} {
				parts := []string{`"Only":{"const":{"fixed":1}}`, `"Rows":` + leaf, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`}
				if forward {
					parts[0], parts[2] = parts[2], parts[0]
				}
				_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
				if count == 45 && err != nil || count == 46 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
					t.Fatal("structured const erased source hops", count, tail, forward, err)
				}
			}
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[{"const":{"fixed":1}}],"default":[{"fixed":1}]}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("structured tuple slot budget", width, err)
		}
	}
}

func TestTupleStructuredConstSources(t *testing.T) {
	for _, tc := range []struct{ member, tail, want string }{
		{`{"const":{"fixed":1}}`, "", "List<kotlinx.serialization.json.JsonObject>"},
		{`{"type":"array","const":[1]}`, "", "List<kotlinx.serialization.json.JsonArray>"},
		{`{"const":{"fixed":1},"nullable":true}`, "", "List<kotlinx.serialization.json.JsonObject?>"},
		{`{"const":[1]}`, `,"items":true`, "List<kotlinx.serialization.json.JsonElement>"},
		{`{"const":[1]}`, `,"items":{"type":"integer"}`, "List<kotlin.Int>"},
		{`{"$ref":"#/components/schemas/Alias"}`, "", "List<kotlinx.serialization.json.JsonObject>"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Only":{"const":{"fixed":1}},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[`+tc.member+`]`+tc.tail+`}}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(tc, checked, err)
			}
			if !strings.Contains(source(t, got, "RowsDto"), tc.want) {
				t.Fatal(tc, source(t, got, "RowsDto"))
			}
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("structured tuple nondeterminism", e)
			}
		}
	}
}

func TestTupleStructuredConstOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		member, value string
		valid         bool
	}{
		{`{"const":{"fixed":1}}`, `[{"fixed":1.0}]`, true},
		{`{"const":{"fixed":1}}`, `[{"fixed":2}]`, false},
		{`{"const":{"a":1,"b":2}}`, `[{"b":2,"a":1}]`, true},
		{`{"const":[1,2]}`, `[[1.0,2e0]]`, true},
		{`{"const":[1,2]}`, `[[2,1]]`, false},
		{`{"const":{"n":9007199254740993}}`, `[{"n":9007199254740992}]`, false},
		{`{"const":{"fixed":1}}`, `[null]`, false},
		{`{"const":{"fixed":1},"nullable":true}`, `[null,{"fixed":1}]`, true},
		{`{"const":{"allOf":[{"$ref":"#/data-only"}],"properties":{"default":{"enum":[1],"const":{"items":true}}},"description":"data"}}`, `[{"description":"data","properties":{"default":{"const":{"items":true},"enum":[1.0]}},"allOf":[{"$ref":"#/data-only"}]}]`, true},
	} {
		for _, root := range []bool{false, true} {
			tuple := `{"type":"array","prefixItems":[` + tc.member + `],"default":` + tc.value + `}`
			owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+owner+`}`), "Dto"))
			if (err == nil) != tc.valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, root, err)
			}
		}
	}
}

func TestTupleStructuredConstInCollectionTupleDefaults(t *testing.T) {
	for _, value := range []string{`[[{"fixed":1}]]`, `[[{"fixed":2}]]`} {
		input := schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"array","items":{"const":{"fixed":1}}}],"default":` + value + `}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if (err == nil) != (value == `[[{"fixed":1}]]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("nested collection borrowed a structured const kind without pin validation", value, err)
		}
	}
}
