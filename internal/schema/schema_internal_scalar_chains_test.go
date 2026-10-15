package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestSchemaInternalScalarChainBorrowedIdentityAndNullability(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"Source":{"properties":{"leaf":{"type":["string","null"]},"false":{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false},"inherit":{"$ref":"#/components/schemas/Source/properties/false"},"true":{"$ref":"#/components/schemas/Source/properties/inherit","nullable":true}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original := get(get(get(root, "components"), "schemas"), "Source")
	a := analyzer{root: root, components: map[string]*component{"Source": {Schema: original, Wire: "Source", Pointer: "/components/schemas/Source"}}}
	for _, tc := range []struct {
		key      string
		steps    int
		nullable bool
	}{{"false", 2, false}, {"inherit", 3, false}, {"true", 4, true}, {"leaf", 0, true}} {
		target, err := a.resolveSchema("#/components/schemas/Source/properties/"+tc.key, "/use")
		if err != nil {
			t.Fatal(tc, err)
		}
		if target.Schema != get(get(original, "properties"), "leaf") || target.Pointer != "/components/schemas/Source/properties/leaf" || target.Catalog != nil || target.Steps != tc.steps || len(a.components) != 1 {
			t.Fatal("lost terminal node/source/height or synthesized nominal identity", tc, target)
		}
		nullable, err := target.effectiveNullable()
		if err != nil || nullable != tc.nullable {
			t.Fatal(tc, nullable, err)
		}
		for _, override := range []*bool{nil, new(false), new(true)} {
			ty, err := a.referenceType(target, override, 1, "/use")
			want := tc.nullable
			if override != nil {
				want = *override
			}
			if err != nil || (ty.Kind == Nullable) != want {
				t.Fatal("use-site override", tc, ty, err)
			}
		}
		// Reusing a resolved target never consumes or resets its height.
		_, err = a.referenceType(target, nil, MaxSchemaDepth, "/later")
		if err == nil || err.Code != "E_SCHEMA_DEPTH" || *err.Pointer != "/later" {
			t.Fatal("reused height", err)
		}
	}
}
