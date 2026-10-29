package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const tupleHierarchyCatalog = `"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"},{"$ref":"#/components/schemas/Other"}],"discriminator":{"propertyName":"kind","mapping":{"group":"#/components/schemas/Group","other":"#/components/schemas/Other"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"sub","mapping":{"first":"#/components/schemas/First","second":"#/components/schemas/Second"}}},"First":{"properties":{"id":{"type":"string"}},"required":["id"]},"Second":{"properties":{"name":{"type":"string"}},"required":["name"]},"Other":{"properties":{"value":{"type":"string"}},"required":["value"]},"Nullable":{"$ref":"#/components/schemas/Root","nullable":true}`

func TestTupleHierarchyReferences(t *testing.T) {
	for _, target := range []string{"Root", "Group", "First", "Nullable"} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{`+tupleHierarchyCatalog+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/`+target+`"}]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(target, tail, checked, err)
				}
				text := source(t, got, "RowsDto")
				want := "List<org.example.models.models." + target + "Dto>"
				if target == "Nullable" {
					want = "List<org.example.models.models.RootDto?>"
				}
				if tail != "" {
					want = "List<kotlin.Int>"
				}
				if !strings.Contains(text, want) {
					t.Fatal("lost actual hierarchy classifier", target, text)
				}
				again, err := core.GenerateWithDiagnostics(rq)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("hierarchy tuple nondeterminism", err)
				}
			}
		}
	}
}

func TestTupleHierarchyOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		target, value string
		valid         bool
	}{
		{"Root", `[{"kind":"group","sub":"first","id":"ready"}]`, true},
		{"Group", `[{"kind":"group","sub":"second","name":"ready"}]`, true},
		{"First", `[{"id":"ready"}]`, true},
		{"Nullable", `[null,{"kind":"other","value":"ready"}]`, true},
		{"Root", `[null]`, false},
		{"First", `[{"id":"ready","kind":"other"}]`, false},
		{"Group", `[{"kind":"other","sub":"first","id":"ready"}]`, false},
		{"Root", `[{"kind":"group","sub":"first"}]`, false},
		{"Root", `[{"kind":"future","id":"ready"}]`, false},
		{"Group", `[{"kind":"group","sub":"future","id":"ready"}]`, false},
	} {
		for _, root := range []bool{false, true} {
			tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + tc.target + `"}],"default":` + tc.value + `}`
			owner := `"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+tupleHierarchyCatalog+`,`+owner+`}`), "Dto"))
			if (err == nil) != tc.valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, root, err)
			}
		}
	}
}

func TestTupleHierarchyAliasNullViewsAndSourceGuards(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, value := range []string{`[null]`, `[{"kind":"other","value":"ready"}]`} {
			member := fmt.Sprintf(`{"$ref":"#/components/schemas/Nullable","nullable":%v}`, nullable)
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+tupleHierarchyCatalog+`,"Rows":{"type":"array","prefixItems":[`+member+`],"default":`+value+`}}`), "Dto"))
			if (err == nil) != (nullable || value != `[null]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(nullable, value, err)
			}
		}
	}
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
		for _, tc := range []struct{ member, code, pointer string }{
			{`{"$ref":"#/components/schemas/Root","default":[]}`, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/prefixItems/0/default"},
			{`{"$ref":"#/components/schemas/Root","nullable":"bad"}`, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/nullable"},
		} {
			failure(t, schemas(`{`+tupleHierarchyCatalog+`,"Rows":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`}}`), tc.code, tc.pointer)
		}
	}
}

func TestTupleHierarchyDepthAndWidth(t *testing.T) {
	for _, target := range []string{"Root", "Group", "First", "Nullable"} {
		boundary := 0
		for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
			last := 0
			for depth := 40; depth <= 49; depth++ {
				leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]` + tail + `}`
				for i := 1; i < depth; i++ {
					leaf = `{"type":"array","items":` + leaf + `}`
				}
				input := schemas(`{` + tupleHierarchyCatalog + `,"Rows":` + leaf + `,"Consumer":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`)
				_, err := core.Generate(request(input, "Dto"))
				if err == nil {
					last = depth
				} else if err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal("hierarchy nominal/alias source depth", target, depth, tail, err)
				}
			}
			if last < 40 || last > 48 {
				t.Fatal("missing bounded hierarchy source depth", target, last)
			}
			if boundary == 0 {
				boundary = last
			} else if last != boundary {
				t.Fatal("projected hierarchy reference erased cached source work", target, tail, boundary, last)
			}
		}
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%03d":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/First"}]}`, i)
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+tupleHierarchyCatalog+`,"Wide":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(width, err)
		}
	}
}

func TestTupleHierarchyCompoundAndClosedDefaults(t *testing.T) {
	compound := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"}],"discriminator":{"propertyName":"family","mapping":{"group":"#/components/schemas/Group"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/A"},{"$ref":"#/components/schemas/B"},{"$ref":"#/components/schemas/C"}]},"A":{"properties":{"kind":{"const":"note"},"mode":{"const":"fast"},"payload":{"type":"string"}},"required":["payload"]},"B":{"properties":{"kind":{"const":"note"},"mode":{"const":"slow"},"payload":{"type":"string"}},"required":["payload"]},"C":{"properties":{"kind":{"const":"count"},"mode":{"const":"fast"},"n":{"type":"integer"}},"required":["n"]}`
	for _, value := range []string{`[{"family":"group","kind":"note","mode":"fast","payload":"ready"}]`, `[{"family":"group","kind":"note","mode":"slow","payload":"ready"}]`, `[{"family":"group","kind":"note","mode":"unknown","payload":"ready"}]`, `[{"family":"group","kind":"note","mode":"fast"}]`} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+compound+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Root"}],"default":`+value+`}}`), "Dto"))
		valid := strings.Contains(value, `"payload"`) && !strings.Contains(value, `"unknown"`)
		if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("compound hierarchy annotation", value, err)
		}
	}
	closed := `"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"}],"discriminator":{"propertyName":"family","mapping":{"group":"#/components/schemas/Group"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/Left"},{"$ref":"#/components/schemas/Right"}]},"Left":{"properties":{"v":{"type":"string"}},"required":["v"]},"Right":{"properties":{"v":{"type":"integer"}},"required":["v"]}`
	for _, value := range []string{`[{"family":"group","type":"left","v":"ready"}]`, `[{"family":"group","type":"right","v":7}]`, `[{"family":"group","v":"ready"}]`, `[{"family":"group","type":"left","v":7}]`} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+closed+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Root"}],"default":`+value+`}}`), "Dto"))
		valid := strings.Contains(value, `"type":"left","v":"ready"`) || strings.Contains(value, `"type":"right","v":7`)
		if (err == nil) != valid || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("closed inner hierarchy annotation", value, err)
		}
	}
}

func TestHierarchyClosedDescendantKeepsActualSubtypeCodec(t *testing.T) {
	input := schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Group"}],"discriminator":{"propertyName":"family","mapping":{"group":"#/components/schemas/Group"}}},"Group":{"oneOf":[{"$ref":"#/components/schemas/Left"},{"$ref":"#/components/schemas/Right"}]},"Left":{"properties":{"v":{"type":"string"}},"required":["v"]},"Right":{"properties":{"v":{"type":"integer"}},"required":["v"]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	text := source(t, got, "RootDto")
	if !strings.Contains(text, "is org.example.models.models.GroupDto -> encoder.encodeSerializableValue(org.example.models.models.GroupDto.serializer(), value)") {
		t.Fatal("outer hierarchy discarded actual inner sealed serialization route")
	}
}

func TestTupleHierarchyOverlappingSourceBudget(t *testing.T) {
	for _, count := range []int{1, 100} {
		for _, positional := range []bool{false, true} {
			catalog := map[string]any{}
			if err := json.Unmarshal([]byte(`{`+tupleHierarchyCatalog+`}`), &catalog); err != nil {
				t.Fatal(err)
			}
			fields := catalog["First"].(map[string]any)["properties"].(map[string]any)
			for i := 0; i < 50; i++ {
				fields[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "string"}
			}
			for i := 0; i < count; i++ {
				member := map[string]any{"$ref": "#/components/schemas/Root"}
				array := map[string]any{"type": "array", "items": member}
				if positional {
					array["prefixItems"], array["items"] = []any{member}, true
				}
				catalog[fmt.Sprintf("Rows%03d", i)] = array
			}
			input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
			if err != nil {
				t.Fatal(err)
			}
			got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if count == 100 && positional {
				if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("overlapping hierarchy source escaped cumulative budget", err)
				}
			} else if err != nil {
				t.Fatal("small tuple/original ordinary hierarchy collection control", count, positional, err)
			}
		}
	}
}
