package core_test

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleBooleanTails(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, element string }{
		{`[{"type":"string"}]`, "false", "kotlin.String"},
		{`[{"type":"string"}]`, "true", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"string"},{"type":"integer"}]`, "false", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"string","format":"date"}]`, "false", "kotlinx.datetime.LocalDate"},
		{`[{"type":["string","null"]}]`, "false", "kotlin.String?"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Rows":{"type":"array","prefixItems":`+tc.prefix+`,"items":`+tc.tail+`}}`), "Dto")
			rq.Options.Validation = checked
			report, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !strings.Contains(source(t, report, "RowsDto"), "List<"+tc.element+">") {
				t.Fatal(tc, checked, err)
			}
			if len(report.Warnings) != 1 || report.Warnings[0].Code != "W_TUPLE_APPROXIMATION" || *report.Warnings[0].Pointer != "/components/schemas/Rows/prefixItems" {
				t.Fatal("Boolean tail lost approximation/source", report.Warnings)
			}
		}
	}
}

func TestTupleBooleanTailsExistingNominalSources(t *testing.T) {
	for _, terminal := range []string{
		`{"type":"string"}`,
		`{"type":"string","enum":["ready","done"]}`,
		`{"type":"integer","enum":[1,2]}`,
		`{"properties":{"value":{"type":"string"}},"required":["value"]}`,
		`{"allOf":[{"properties":{"value":{"type":"string"}}}]}`,
	} {
		for _, tail := range []string{"false", "true"} {
			rq := request(schemas(`{"Leaf":`+terminal+`,"Alias":{"$ref":"#/components/schemas/Leaf"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Leaf"}],"items":`+tail+`}}`), "Dto")
			rq.Options.Validation = true
			report, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal(terminal, tail, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(report, again) {
				t.Fatal("Boolean tail changed nominal determinism", err)
			}
		}
	}
}

func TestTupleBooleanTailSourceErrorsPrecedeProjection(t *testing.T) {
	for _, tail := range []string{"true", "false"} {
		for _, tc := range []struct{ member, pointer string }{
			{`{"type":"string","format":3}`, "/components/schemas/Rows/prefixItems/0"},
			{`{"type":"integer","unknown":true}`, "/components/schemas/Rows/prefixItems/0/unknown"},
		} {
			rq := request(schemas(`{"Rows":{"type":"array","prefixItems":[`+tc.member+`],"items":`+tail+`}}`), "Dto")
			report, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("Boolean tail hid invalid original prefix", tail, tc, err)
			}
		}
	}
}

func TestTupleBooleanTailFieldAndRootDefaults(t *testing.T) {
	for _, required := range []string{"", `,"required":["values"]`} {
		input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"integer"}],"items":false,"default":[]},"Holder":{"properties":{"values":{"type":"array","prefixItems":[{"type":"integer"}],"items":false,"default":[1]}}` + required + `}}`)
		if _, err := core.Generate(request(input, "Dto")); err != nil {
			t.Fatal("closed-tail selected scalar lost existing default role", err)
		}
		bad := strings.Replace(string(input), `"default":[1]`, `"default":[true]`, 1)
		failure(t, []byte(bad), "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/values/default/0")
	}
}

func TestTupleBooleanTailRetainsOriginalCollectionChecks(t *testing.T) {
	for _, tail := range []string{"true", "false"} {
		rq := request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"integer"}],"items":`+tail+`,"minItems":1,"maxItems":3}}}}`), "Dto")
		if _, err := core.GenerateWithDiagnostics(rq); err == nil {
			t.Fatal("Boolean tail removed validation opt-in")
		}
		rq.Options.Validation = true
		if _, err := core.GenerateWithDiagnostics(rq); err != nil {
			t.Fatal("Boolean tail lost existing counts", tail, err)
		}
	}
	// A true source already owns the exact JSON-value uniqueness profile;
	// false does not authorize a projected/raw or positional uniqueness source.
	rq := request(schemas(`{"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"type":"integer"}],"items":true,"uniqueItems":true}}}}`), "Dto")
	rq.Options.Validation = true
	if _, err := core.GenerateWithDiagnostics(rq); err != nil {
		t.Fatal("true tail lost existing JSON-value uniqueness", err)
	}
	rq.Document = []byte(strings.Replace(string(rq.Document), `"items":true`, `"items":false`, 1))
	if report, err := core.GenerateWithDiagnostics(rq); err == nil || len(report.Artifacts.Files) != 0 || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/rows/uniqueItems" {
		t.Fatal("false tail invented positional uniqueness authority", err)
	}
}

func TestTupleBooleanTailOriginalCorpusMigration(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/p1/valid/boolean-tuple-tail.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := core.Generate(request(input, "")); err != nil {
		t.Fatal("exact former corpus refusal remains", err)
	}
	for _, tail := range []bool{false, true} {
		var nested = fmt.Sprintf(`{"type":"array","prefixItems":[{"type":"integer"}],"items":%t}`, tail)
		for depth := 1; depth <= 49; depth++ {
			rq := request(schemas(`{"Rows":`+nested+`}`), "Dto")
			report, err := core.GenerateWithDiagnostics(rq)
			if depth == 48 && err != nil || depth == 49 && (err == nil || len(report.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
				t.Fatal("Boolean tail reset source depth", tail, depth, err)
			}
			nested = `{"type":"array","items":` + nested + `}`
		}
	}
}
