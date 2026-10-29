package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

const tupleInlineAllOfItem = `{"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},{"properties":{"n":{"type":"integer","default":7}}}]}`

func TestTupleInlineAllOfSourcesAndOwnDefaults(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, value, want string }{
		{`[` + tupleInlineAllOfItem + `]`, "", `[{"name":"root"}]`, "RowsDtoInner>"},
		{`[` + tupleInlineAllOfItem + `]`, `,"items":false`, `[]`, "RowsDtoInner>"},
		{`[` + tupleInlineAllOfItem + `]`, `,"items":true`, `[null,{"default":"data"}]`, "JsonElement>"},
		{`[` + tupleInlineAllOfItem + `,` + tupleInlineAllOfItem + `]`, "", `[null,1]`, "JsonElement>"},
		{`[false,{"type":"integer"}]`, `,"items":` + tupleInlineAllOfItem, `[{"name":"tail"}]`, "RowsDtoInner>"},
	} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				tuple := `{"type":"array","prefixItems":` + tc.prefix + tc.tail + `,"default":` + tc.value + `}`
				input := schemas(`{"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":` + tuple + `}}}`)
				rq := request(input, "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(tc, style, checked, err)
				}
				if !strings.Contains(source(t, got, "RowsDto"), tc.want) {
					t.Fatal(tc, source(t, got, "RowsDto"))
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("inline allOf tuple nondeterminism", e)
				}
			}
		}
	}
}

func TestTupleInlineAllOfSourceCannotHideDeferredDefaults(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"string"}`} {
		input := schemas(`{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[{"allOf":[{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"future"}}},{"properties":{"name":{"type":"string"}}}]}]` + tail + `,"default":[]}}`)
		failure(t, input, "E_SCHEMA_INVALID", "/components/schemas/Rows/prefixItems/0/allOf/0/properties/state/default")
		// Keep a real constructor field beside the pinned enum body so this
		// checks ordinary source diagnostics rather than const-only admission.
		overridden := `{"allOf":[{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"future"}}},{"properties":{"state":{"$ref":"#/components/schemas/Choice","default":"ready"},"name":{"type":"string"}}}]}`
		_, ordinary := core.Generate(request(schemas(`{"Choice":{"enum":["ready"]},"Item":`+overridden+`}`), "Dto"))
		if ordinary == nil {
			t.Fatal("ordinary source hid an invalid overwritten annotation")
		}
		original := ordinary.(core.Diagnostics)[0]
		pointer := strings.Replace(*original.Pointer, "/components/schemas/Item", "/components/schemas/Rows/prefixItems/0", 1)
		input = schemas(`{"Choice":{"enum":["ready"]},"Rows":{"type":"array","prefixItems":[` + overridden + `]` + tail + `}}`)
		_, actual := core.Generate(request(input, "Dto"))
		if actual == nil {
			t.Fatal("tuple projection hid invalid source override")
		}
		diagnostic := actual.(core.Diagnostics)[0]
		if diagnostic.Code != original.Code || *diagnostic.Pointer != pointer {
			t.Fatalf("ordinary source diagnostic changed: want %s at %s, got %s at %s", original.Code, pointer, diagnostic.Code, *diagnostic.Pointer)
		}
	}
	for _, item := range []string{`{"allOf":[{}]}`, `{"allOf":[{"properties":{"pin":{"const":"only"}}}]}`, `{"allOf":[{"additionalProperties":{"type":"string"}}]}`, `{"allOf":[{"oneOf":[{"type":"string"},{"type":"integer"}]}]}`} {
		for _, tail := range []string{"", `,"items":true`} {
			got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":{"type":"array","prefixItems":[`+item+`]`+tail+`}}`), "Dto"))
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("inline allOf manufactured constructor", item, tail, err)
			}
		}
	}
}

func TestTupleInlineAllOfDefaultCounts(t *testing.T) {
	item := strings.TrimSuffix(tupleInlineAllOfItem, "}") + `,"minProperties":2}`
	for _, value := range []string{`[{"name":"outer"}]`, `[{"name":"outer","n":1}]`} {
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[`+item+`],"default":`+value+`}}`), "Dto")
		rq.Options.Validation = true
		_, err := core.Generate(rq)
		if strings.Contains(value, `"n"`) && err != nil || !strings.Contains(value, `"n"`) && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0") {
			t.Fatal("inline allOf count default", value, err)
		}
	}
}

func TestTupleInlineAllOfMemberNullIntersection(t *testing.T) {
	for _, second := range []bool{false, true} {
		item := `{"allOf":[{"properties":{"name":{"type":"string"}},"nullable":true},{"properties":{"n":{"type":"integer"}},"nullable":` + map[bool]string{false: "false", true: "true"}[second] + `}]}`
		rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[`+item+`],"default":[null]}}`), "Dto")
		_, err := core.Generate(rq)
		if second && err != nil || !second && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Rows/default/0") {
			t.Fatal("inline allOf member null intersection", second, err)
		}
	}
}

func TestTupleInlineAllOfOverlappingSourceBudget(t *testing.T) {
	fields := map[string]any{}
	for i := 0; i < 50; i++ {
		fields[fmt.Sprintf("p%03d", i)] = map[string]any{"type": "string"}
	}
	for _, tuple := range []bool{false, true} {
		catalog := map[string]any{"Base": map[string]any{"properties": fields}}
		for i := 0; i < 100; i++ {
			item := map[string]any{"allOf": []any{map[string]any{"$ref": "#/components/schemas/Base"}, map[string]any{"properties": map[string]any{"extra": map[string]any{"type": "integer"}}}}}
			array := map[string]any{"type": "array", "items": item}
			if tuple {
				array["prefixItems"], array["items"] = []any{item}, true
			}
			catalog[fmt.Sprintf("Rows%03d", i)] = array
		}
		input, e := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": catalog}})
		if e != nil {
			t.Fatal(e)
		}
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if !tuple && err != nil {
			t.Fatal("new tuple budget changed existing ordinary allOf elements", err)
		}
		if tuple && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("overlapping inline tuple source snapshots reset cumulative budget", err)
		}
	}
}

func TestTupleInlineAllOfSourceDepthAndConstructorWidth(t *testing.T) {
	for _, count := range []int{46, 47} {
		item := `{"type":"array","prefixItems":[` + tupleInlineAllOfItem + `],"items":true}`
		for i := 1; i < count; i++ {
			item = `{"type":"array","items":` + item + `}`
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Rows":`+item+`}`), "Dto"))
		if count == 46 && err != nil || count == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("discarded inline allOf source depth", count, err)
		}
	}
	for _, count := range []int{44, 45} {
		item := `{"type":"array","prefixItems":[` + tupleInlineAllOfItem + `],"items":true}`
		for i := 1; i < count; i++ {
			item = `{"type":"array","items":` + item + `}`
		}
		for _, forward := range []bool{false, true} {
			parts := []string{`"Rows":` + item, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`}
			if forward {
				parts[0], parts[1] = parts[1], parts[0]
			}
			_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
			if count == 44 && err != nil || count == 45 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
				t.Fatal("discarded inline allOf cached incoming source depth", count, forward, err)
			}
		}
	}
	for _, width := range []int{240, 241} {
		properties := make([]string, width)
		for i := range properties {
			properties[i] = fmt.Sprintf(`"v%03d":{"type":"array","prefixItems":[%s],"default":[]}`, i, tupleInlineAllOfItem)
		}
		_, err := core.Generate(request(schemas(`{"Holder":{"properties":{`+strings.Join(properties, ",")+`}}}`), "Dto"))
		if width == 240 && err != nil || width == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("inline allOf JVM slot limit", width, err)
		}
	}
}
