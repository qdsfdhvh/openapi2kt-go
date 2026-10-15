package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestSchemaInternalCollectionChainIdentityNullabilityAndCycles(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Source":{"properties":{"leaf":{"type":["string","null"]},"list":{"type":["array","null"],"items":{"$ref":"#/components/schemas/Source/properties/leaf"}},"falseHop":{"$ref":"#/components/schemas/Source/properties/list","nullable":false},"trueHop":{"$ref":"#/components/schemas/Source/properties/falseHop","nullable":true},"map":{"additionalProperties":{"$ref":"#/components/schemas/Source/properties/falseHop"}},"cycle":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/cycleHop"}},"cycleHop":{"$ref":"#/components/schemas/Source/properties/cycle"}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	owner := get(get(get(root, "components"), "schemas"), "Source")
	a := analyzer{root: root, components: map[string]*component{"Source": {Schema: owner, Wire: "Source", Pointer: "/components/schemas/Source"}}}
	props := get(owner, "properties")
	for _, tc := range []struct {
		key      string
		steps    int
		nullable bool
	}{
		{"falseHop", 2, false}, {"trueHop", 3, true},
	} {
		target, err := a.resolveSchema("#/components/schemas/Source/properties/"+tc.key, "/use")
		if err != nil || target.Catalog != nil || target.Schema != get(props, "list") || target.Steps != tc.steps || target.Nullable == nil || *target.Nullable != tc.nullable || len(a.components) != 1 {
			t.Fatal("chain lost original terminal identity/nearest nullable", tc, target, err)
		}
	}
	for _, key := range []string{"cycle", "cycleHop"} {
		_, err := a.resolveSchema("#/components/schemas/Source/properties/"+key, "/use")
		if err == nil || err.Code != "E_REF_UNSUPPORTED" || *err.Pointer != "/components/schemas/Source/properties/cycle/items" {
			t.Fatal("container/ref cycle classification", key, err)
		}
	}
	// Failed admission cannot poison a later independent finite use.
	target, err := a.resolveSchema("#/components/schemas/Source/properties/map", "/use")
	if err != nil || target.Schema != get(props, "map") || target.Catalog != nil {
		t.Fatal("independent finite admission", target, err)
	}
}
