package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleUnionSourceDepthAndWidth(t *testing.T) {
	for _, referenced := range []bool{false, true} {
		var boundary int
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			last := 0
			for depth := 40; depth <= 49; depth++ {
				member := `{"oneOf":[{"type":"string"},{"type":"integer"}]}`
				if referenced {
					member = `{"$ref":"#/components/schemas/Choice"}`
				}
				leaf := `{"type":"array","prefixItems":[` + member + `]` + tail + `}`
				for i := 1; i < depth; i++ {
					leaf = `{"type":"array","items":` + leaf + `}`
				}
				catalog := `"Choice":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Rows":` + leaf + `,"Consumer":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`
				_, err := core.Generate(request(schemas(`{`+catalog+`}`), "Dto"))
				if err == nil {
					last = depth
				} else if err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal("unexpected union-depth failure", depth, tail, err)
				}
			}
			if last < 40 || last > 48 {
				t.Fatal("missing bounded union source depth", last)
			}
			if boundary == 0 {
				boundary = last
			} else if last != boundary {
				t.Fatal("discarded union source evaded cached source depth", referenced, tail, boundary, last)
			}
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[{"oneOf":[{"type":"string"},{"type":"integer"}]}]}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, err)
		}
	}
}

func TestTupleUnionSources(t *testing.T) {
	for _, union := range []string{
		`{"oneOf":[{"type":"string"},{"type":"integer"}]}`,
		`{"oneOf":[{"properties":{"name":{"type":"string"}}},{"properties":{"n":{"type":"integer"}}}]}`,
	} {
		for _, reference := range []bool{false, true} {
			for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
				for _, checked := range []bool{false, true} {
					member := union
					if reference {
						member = `{"$ref":"#/components/schemas/Only"}`
					}
					rq := request(schemas(`{"Only":`+union+`,"Rows":{"type":"array","prefixItems":[`+member+`]`+tail+`}}`), "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(union, reference, tail, checked, err)
					}
					text := source(t, got, "RowsDto")
					if tail == "" && !strings.Contains(text, "List<") || tail != "" && !strings.Contains(text, "List<kotlin.Int>") {
						t.Fatal("tuple lost its selected codec", text)
					}
					if tail == "" && strings.Contains(text, "List<kotlinx.serialization.json.JsonElement>") {
						t.Fatal("nominal union was replaced with a raw carrier", text)
					}
					again, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("union tuple nondeterminism", err)
					}
				}
			}
		}
	}
}

func TestTupleUnionDistinctInlineSourcesRemainMixed(t *testing.T) {
	union := `{"oneOf":[{"type":"string"},{"type":"integer"}]}`
	got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","prefixItems":[`+union+`,`+union+`]}}`), "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "RowsDto"), "List<kotlinx.serialization.json.JsonElement>") {
		t.Fatal("separate union nodes acquired shared nominal identity")
	}
}

func TestTupleUnionDiscardedSourceDefaultsStillValidate(t *testing.T) {
	for _, tail := range []string{`,"items":true`, `,"items":{"type":"integer"}`} {
		input := schemas(`{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[{"oneOf":[{"properties":{"a":{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"bad"}}}}},{"properties":{"b":{"type":"integer"}}}]}]` + tail + `}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("discarded union bypassed its original deferred default", err)
		}
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/oneOf/0/properties/a/properties/state/default")
	}
}

func TestTupleUnionOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		union, value string
		valid        bool
	}{
		{`{"oneOf":[{"type":"string"},{"type":"integer"}]}`, `["ready",1]`, true},
		{`{"oneOf":[{"type":"string"},{"type":"integer"}]}`, `[true]`, false},
		{`{"oneOf":[{"type":"string","const":"ready"},{"type":"integer","enum":[1,2]}]}`, `["ready",2]`, true},
		{`{"oneOf":[{"type":"string","const":"ready"},{"type":"integer","enum":[1,2]}]}`, `["other"]`, false},
		{`{"oneOf":[{"type":"string"},{"type":"integer"}],"nullable":true}`, `[null,"ready"]`, true},
		{`{"oneOf":[{"type":"integer"},{"type":"integer"}]}`, `[1]`, true},
		{`{"oneOf":[{"type":"string","enum":["ready"]},{"type":"string"}]}`, `["ready"]`, true},
		{`{"oneOf":[{"type":"string","enum":["ready"]},{"type":"string"}]}`, `["unknown"]`, false},
		{`{"oneOf":[{"type":"integer"},{"type":"integer"}]}`, `["bad"]`, false},
		{`{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"}},"required":["name"]},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}},"required":["n"]}]}`, `[{"kind":"text","name":"ready"}]`, true},
		{`{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"}},"required":["name"]},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}},"required":["n"]}]}`, `[{"kind":"text"}]`, false},
		{`{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"}}},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}}}]}`, `[{"kind":"unknown"}]`, false},
	} {
		for _, reference := range []bool{false, true} {
			if reference && (tc.union == `{"oneOf":[{"type":"integer"},{"type":"integer"}]}` || strings.Contains(tc.union, `"enum":["ready"]`)) {
				continue
			}
			for _, root := range []bool{false, true} {
				member := tc.union
				if reference {
					member = `{"$ref":"#/components/schemas/Only"}`
				}
				tuple := `{"type":"array","prefixItems":[` + member + `],"default":` + tc.value + `}`
				owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
				if !root {
					owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
				}
				if reference {
					owner = `"Only":` + tc.union + `,` + owner
				}
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+owner+`}`), "Dto"))
				if (err == nil) != tc.valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
					if err != nil {
						t.Log("pointer", *err.(core.Diagnostics)[0].Pointer)
					}
					t.Fatal(tc, reference, root, err)
				}
			}
		}
	}
}

func TestTupleUnionClosedRootDefaults(t *testing.T) {
	for _, root := range []bool{false, true} {
		for _, value := range []string{`[{"type":"Variant1","value":"ready"}]`, `[{"type":"Variant1","value":false}]`, `[{"value":"ready"}]`} {
			tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Choice"}],"default":` + value + `}`
			owner := `"Rows":` + tuple
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `}}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Choice":`+closedUnionDefaultTarget+`,`+owner+`}`), "Dto"))
			if (err == nil) != (value == `[{"type":"Variant1","value":"ready"}]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(root, value, err)
			}
		}
	}
}

func TestTupleUnionOriginalSourceGuards(t *testing.T) {
	for _, member := range []string{
		`{"oneOf":[{"type":"string"},{"type":"integer"}],"default":"ready"}`,
		`{"oneOf":[{"type":"string","nullable":true},{"type":"integer"}]}`,
		`{"oneOf":[{"type":"string"},{"type":"integer"}],"unknownRole":true}`,
		`{"oneOf":[{"type":"string"},{"type":"array","items":{"type":"integer"}}]}`,
	} {
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","prefixItems":[`+member+`]`+tail+`}}`), "Dto"))
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("unsupported source hidden by projection", member, tail, err)
			}
		}
	}
	for _, count := range []int{240, 241} {
		branches := make([]string, count)
		for i := range branches {
			branches[i] = `{"type":"integer"}`
		}
		input := schemas(`{"Rows":{"type":"array","prefixItems":[{"oneOf":[` + strings.Join(branches, ",") + `]}],"items":true}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("discarded union branch cap", count, err)
		}
	}
}

func TestTupleUnionLiteralPayloadsAreData(t *testing.T) {
	union := `{"oneOf":[{"properties":{"kind":{"const":"text"},"name":{"type":"string"},"data":{"const":{"allOf":[{"$ref":"#/data-only"}],"description":"data"}}}},{"properties":{"kind":{"const":"count"},"n":{"type":"integer"}}}]}`
	if _, err := core.Generate(request(schemas(`{"Choice":`+union+`}`), "Dto")); err != nil {
		t.Fatal("original union control", err)
	}
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
		if _, err := core.Generate(request(schemas(`{"Choice":`+union+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Choice"}]`+tail+`}}`), "Dto")); err != nil {
			t.Fatal("literal JSON data acquired schema authority", tail, err)
		}
	}
}
