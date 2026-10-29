package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

// Keep former exact refusals as lexical-codec positives. Metadata never grants
// their temporal/UUID/provider counterparts' parsing or equality semantics.
func metadataStringEnumGraduation(t *testing.T, rq core.GenerationRequest) {
	t.Helper()
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("eligible metadata enum remains rejected", err)
	}
	lexical := false
	for _, file := range got.Artifacts.Files {
		lexical = lexical || strings.Contains(file.Content, "public abstract val serialName: kotlin.String") && strings.Contains(file.Content, "decoder.decodeString()")
	}
	if !lexical {
		t.Fatal("metadata lost its actual lexical enum codec")
	}
}

func TestMetadataStringEnumModelRoles(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready","future"],` + metadata + `}`
		nullable := strings.TrimSuffix(leaf, "}") + `,"nullable":true}`
		own := strings.TrimSuffix(leaf, "}") + `,"default":"ready"}`
		for _, tc := range []struct{ name, schemas string }{
			{"root-default", `{"Value":` + own + `}`},
			{"nullable-root", `{"Value":` + strings.TrimSuffix(nullable, "}") + `,"default":null}}`},
			{"aliases", `{"Value":` + own + `,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false,"default":"future"}}`},
			{"annotated-tuple-target", `{"Value":` + own + `,"Alias":{"$ref":"#/components/schemas/Value","default":"future"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"default":["ready"]}}`},
			{"direct-fields", `{"Holder":{"properties":{"state":` + own + `,"required":` + own + `},"required":["required"]}}`},
			{"direct-list", `{"Holder":{"properties":{"rows":{"type":"array","items":` + leaf + `,"default":["ready"]}}}}`},
			{"direct-map", `{"Holder":{"properties":{"rows":{"type":"object","additionalProperties":` + leaf + `,"default":{"first":"ready"}}}}}`},
			{"nested-list-map", `{"Holder":{"properties":{"rows":{"type":"array","items":{"type":"object","additionalProperties":` + nullable + `},"default":[{"first":"ready","second":null}]}}}}`},
			{"named-list-default", `{"Value":` + leaf + `,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"}},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":["ready"]}}}}`},
			{"reference-field", `{"Value":` + leaf + `,"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias","default":"ready"}}}}`},
			{"independent-reference-defaults", `{"Value":` + own + `,"Alias":{"$ref":"#/components/schemas/Value","default":"future"},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias","default":"ready"}}}}`},
			{"internal-reference", `{"Holder":{"properties":{"state":` + leaf + `}},"Alias":{"$ref":"#/components/schemas/Holder/properties/state"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}`},
			{"internal-tuple", `{"Holder":{"properties":{"state":` + leaf + `}},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Holder/properties/state"}],"default":["ready"]}}`},
			{"borrowed-collection", `{"Holder":{"properties":{"rows":{"type":"array","items":` + leaf + `}}},"Rows":{"$ref":"#/components/schemas/Holder/properties/rows"}}`},
			{"object-default", `{"Value":` + leaf + `,"Alias":{"$ref":"#/components/schemas/Value"},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias"}},"required":["state"],"default":{"state":"ready"}}}`},
			{"object-independent-defaults", `{"Value":` + own + `,"Alias":{"$ref":"#/components/schemas/Value","default":"future"},"Holder":{"properties":{"state":{"$ref":"#/components/schemas/Alias"}},"required":["state"],"default":{"state":"ready"}}}`},
		} {
			for _, checked := range []bool{false, true} {
				t.Run(metadata+"/"+tc.name+"/"+map[bool]string{false: "unchecked", true: "checked"}[checked], func(t *testing.T) {
					rq := request(schemas(tc.schemas), "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(err, *err.(core.Diagnostics)[0].Pointer)
					}
					if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("metadata enum nondeterminism", err)
					}
				})
			}
		}
	}
}

func TestMetadataStringEnumLexicalUniqueness(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready","future"],` + metadata + `}`
		for _, items := range []string{leaf, `{"$ref":"#/components/schemas/Value"}`} {
			input := schemas(`{"Value":` + leaf + `,"Holder":{"properties":{"rows":{"type":"array","items":` + items + `,"uniqueItems":true,"default":["ready","future"]}}}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = true
			if _, err := core.Generate(rq); err != nil {
				t.Fatal(metadata, items, err, *err.(core.Diagnostics)[0].Pointer)
			}
			rq.Document = []byte(strings.Replace(string(input), `"default":["ready","future"]`, `"default":["ready","ready"]`, 1))
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/Holder/properties/rows/default/1" {
				t.Fatal("metadata enum uniqueness lost original lexical key", metadata, err)
			}
		}
	}
}

func TestMetadataStringEnumMergedFactories(t *testing.T) {
	rq := request(schemas(`{"A":{"type":"string","format":"date","enum":["ready"],"default":"ready"},"B":{"type":"string","format":"date","enum":["ready"],"default":"ready"},"C":{"type":["string","null"],"format":"date","enum":["ready"],"default":null},"D":{"type":["string","null"],"format":"date","enum":["ready"],"default":null}}`), "Dto")
	rq.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, signature string }{{"BDto", "public fun `BDto default`(): BDto"}, {"DDto", "public fun `DDto default`(): DDto? = null"}} {
		text := source(t, got, tc.name)
		if !strings.Contains(text, "public typealias "+tc.name) || !strings.Contains(text, tc.signature) {
			t.Fatal("merge lost a component-owned enum factory", text)
		}
	}
}

// Select only the newly admitted lexical metadata leaves in historical guard
// tables. Bounds, compositions, pins, value classes and null vocabularies keep
// their original negative tests.
func metadataStringEnumFixture(raw string) bool {
	var v map[string]any
	if json.Unmarshal([]byte(raw), &v) != nil || v["enum"] == nil || (v["format"] == nil && v["x-native-type"] == nil) {
		return false
	}
	for key := range v {
		switch key {
		case "type", "nullable", "format", "enum", "x-native-type", "default":
		default:
			return false
		}
	}
	if kind, ok := v["type"]; ok {
		switch kind := kind.(type) {
		case string:
			if kind != "string" {
				return false
			}
		case []any:
			if len(kind) != 2 || !((kind[0] == "string" && kind[1] == "null") || (kind[1] == "string" && kind[0] == "null")) {
				return false
			}
		default:
			return false
		}
	}
	values, ok := v["enum"].([]any)
	if !ok || len(values) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		wire, ok := value.(string)
		if !ok || seen[wire] {
			return false
		}
		seen[wire] = true
	}
	return true
}
func metadataStringEnumCollectionFixture(raw string) bool {
	var v map[string]any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return false
	}
	for depth := 0; depth < 48; depth++ {
		var next any
		if v["items"] != nil {
			next = v["items"]
		} else if v["additionalProperties"] != nil {
			next = v["additionalProperties"]
		} else {
			data, _ := json.Marshal(v)
			return metadataStringEnumFixture(string(data))
		}
		var ok bool
		v, ok = next.(map[string]any)
		if !ok {
			return false
		}
	}
	return false
}

func TestMetadataStringEnumStrictAnnotationsAndSourceGuards(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready"],` + metadata + `}`
		for _, tc := range []struct{ body, pointer string }{
			{`{"Value":` + strings.TrimSuffix(leaf, "}") + `,"default":"future"}}`, "/components/schemas/Value/default"},
			{`{"Value":` + strings.TrimSuffix(leaf, "}") + `,"default":null}}`, "/components/schemas/Value/default"},
			{`{"Value":` + leaf + `,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false,"default":null}}`, "/components/schemas/Narrow/default"},
			{`{"Holder":{"properties":{"value":` + strings.TrimSuffix(leaf, "}") + `,"default":"future"}},"required":["value"]}}`, "/components/schemas/Holder/properties/value/default"},
			{`{"Value":` + leaf + `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Value","default":"future"}},"required":["value"]}}`, "/components/schemas/Holder/properties/value/default"},
			{`{"Holder":{"properties":{"value":{"type":"array","items":` + leaf + `,"default":["ready","future"]}}}}`, "/components/schemas/Holder/properties/value/default/1"},
		} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(tc.body), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("invalid metadata annotation accepted", metadata, tc, err)
				}
				d := err.(core.Diagnostics)[0]
				if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != tc.pointer {
					t.Fatal(metadata, tc, err)
				}
			}
		}
	}
	for _, leaf := range []string{`{"type":"string","enum":["ready"],"format":"unknown"}`, `{"type":"string","enum":["ready"],"x-native-type":"bad.Type()"}`, `{"type":"string","enum":[null,"ready"],"format":"date","nullable":true}`, `{"type":"string","enum":["ready"],"format":"date","minLength":1}`} {
		if got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+leaf+`}`), "Dto")); err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("metadata source bypassed its original grammar", leaf, err)
		}
	}
}

func TestMetadataStringEnumAliasDepthMatchesPlain(t *testing.T) {
	for count := 43; count <= 49; count++ {
		for _, checked := range []bool{false, true} {
			var accepted bool
			for i, extra := range []string{"", `,"format":"date"`, `,"x-native-type":"org.example.provider.Amount"`} {
				leaf := `{"type":"string","nullable":true,"enum":["ready"]` + extra + `}`
				rq := request(rootSpecialChain(count, leaf, fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, count-1)), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if i == 0 {
					accepted = err == nil
				} else if (err == nil) != accepted {
					t.Fatal("metadata lost original alias depth", count, extra, err)
				}
				if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(count, extra, err)
				}
			}
		}
	}
}
