package core_test

import (
	"encoding/json"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestIntrinsicHierarchyNullabilityIndependentFromMappingEdges(t *testing.T) {
	const nodes = `"First":{"nullable":true,"properties":{"kind":{"type":"string","const":"first"},"id":{"type":"string"}},"required":["id"]},"Second":{"properties":{"kind":{"type":"string","const":"second"},"id":{"type":"string"}},"required":["id"]}`
	for _, mapping := range []string{`{"first":"First","second":"Second"}`, `{"first":"Alias","second":"Second"}`} {
		body := `{"Alias":{"$ref":"#/components/schemas/First","nullable":false},"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":` + mapping + `}},` + nodes + `}`
		got, err := core.GenerateWithDiagnostics(request(schemas(body), "Dto"))
		if err != nil {
			t.Fatal(mapping, err)
		}
		root := source(t, got, "RootDto")
		if !strings.Contains(root, `tag == "first" -> org.example.models.models.FirstDto.serializer()`) {
			t.Fatal(root)
		}
		alias := source(t, got, "AliasDto")
		if strings.Contains(alias, "?") {
			t.Fatal("false didn't narrow", alias)
		}
	}
	for _, tc := range []struct{ body, pointer string }{
		{`{"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/First","nullable":true},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind"}},` + nodes + `}`, "/components/schemas/Root/oneOf/0/nullable"},
		{`{"Alias":{"$ref":"#/components/schemas/First","nullable":true},"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":{"first":"Alias"}}},` + nodes + `}`, "/components/schemas/Root/discriminator/mapping/first"},
	} {
		// The unchanged intrinsic-root input now admits explicit nullable
		// members and nullable mapping aliases without nullable inheritance.
		r := report(t, schemas(tc.body), "Dto")
		if !strings.Contains(source(t, r, "RootDto"), "org.example.models.models.FirstDto.serializer()") || !strings.Contains(source(t, r, "FirstDto"), ") : org.example.models.models.RootDto() {") {
			t.Fatal(tc.pointer, "physical mapping lost")
		}
	}
	// Invalid source checks survive intrinsic intent and aliases.
	var doc map[string]any
	if err := json.Unmarshal(hierarchyIntrinsicNullableInput(t), &doc); err != nil {
		t.Fatal(err)
	}
	doc["components"].(map[string]any)["schemas"].(map[string]any)["First"].(map[string]any)["properties"].(map[string]any)["id"].(map[string]any)["format"] = "unsupported"
	input, _ := json.Marshal(doc)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/First/properties/id/format")
}
