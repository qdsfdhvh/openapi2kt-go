package cli_test

import (
	"encoding/json"
	"reflect"
	"testing"
)

func aliasConsumerDocument(t *testing.T, edges int) []byte {
	t.Helper()
	var alias any = map[string]any{"type": "string"}
	for range edges {
		alias = map[string]any{"type": "array", "items": alias}
	}
	doc := map[string]any{"openapi": "3.0.3", "components": map[string]any{"schemas": map[string]any{
		"ZAlias":   alias,
		"Consumer": map[string]any{"properties": map[string]any{"v": map[string]any{"$ref": "#/components/schemas/ZAlias"}}},
	}}}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sourceAliasConsumerDocument(t *testing.T, edges int) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(aliasConsumerDocument(t, edges), &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	schemas["Leaf"] = map[string]any{"type": "string"}
	schemas["Link"] = map[string]any{"$ref": "#/components/schemas/Leaf"}
	var alias any = map[string]any{"$ref": "#/components/schemas/Link"}
	for range edges {
		alias = map[string]any{"type": "array", "items": alias}
	}
	schemas["ZAlias"] = alias
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCollectionAliasConsumerSourceHeightNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, sourceAliasConsumerDocument(t, 44))
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, sourceAliasConsumerDocument(t, 45))
	for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_DEPTH")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Consumer/properties/v" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("erased scalar source height/failure preservation", ds)
		}
	}
}

func unionAliasConsumerDocument(t *testing.T, edges int) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(aliasConsumerDocument(t, edges), &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	var alias any = map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}}
	for range edges {
		alias = map[string]any{"type": "array", "items": alias}
	}
	schemas["ZAlias"] = alias
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestCollectionAliasConsumerUnionHeightNative(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, unionAliasConsumerDocument(t, 45))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	put(t, p.Input, unionAliasConsumerDocument(t, 46))
	for _, args := range [][]string{nil, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_DEPTH")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Consumer/properties/v" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("normalized union source height/failure preservation", ds)
		}
	}
}

func TestCollectionAliasConsumerNativePreservation(t *testing.T) {
	p := newProject(t)
	put(t, p.Input, aliasConsumerDocument(t, 46))
	result(t, p.generate(t), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--check"), 0)
	result(t, p.generate(t), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("boundary alias no-op changed output")
	}
	put(t, p.Input, aliasConsumerDocument(t, 47))
	for _, args := range [][]string{nil, {"--check"}} {
		ds := fails(t, p.generate(t, args...), 1, "E_SCHEMA_DEPTH")
		if *ds.Diagnostics[0].Pointer != "/components/schemas/Consumer/properties/v" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("consumer-depth pointer/failure preservation", ds)
		}
	}
	fails(t, invoke(t, "", "validate", "--input", p.Input, "--diagnostic-format", "json"), 1, "E_SCHEMA_DEPTH")
}
