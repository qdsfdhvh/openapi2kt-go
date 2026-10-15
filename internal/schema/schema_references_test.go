package schema

import (
	"reflect"
	"testing"

	"openapi-kt/internal/document"
)

func TestSchemaInternalScalarPointerBorrowedIdentityAndRole(t *testing.T) {
	root, err := document.Load([]byte(`{"components":{"schemas":{"S~lot/key":{"type":"object","properties":{"":{"type":"boolean"},"01":{"type":"integer"},"~1":{"type":"string"}},"example":{"type":"object","properties":{"leaf":{"type":"string"}}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	original := get(get(get(root, "components"), "schemas"), "S~lot/key")
	catalog := &component{Schema: original, Pointer: "/components/schemas/S~0lot~1key", Wire: "S~lot/key"}
	a := analyzer{root: root, components: map[string]*component{"S~lot/key": catalog}}
	for _, tc := range []struct{ encoded, wire string }{{"", ""}, {"01", "01"}, {"~01", "~1"}} {
		target, err := a.resolveSchema("#/components/schemas/S~0lot~1key/properties/"+tc.encoded, "/use")
		if err != nil {
			t.Fatal(err)
		}
		if target.Catalog != nil || target.Schema != get(get(original, "properties"), tc.wire) || target.Pointer != "/components/schemas/S~0lot~1key/properties/"+tc.encoded || len(a.components) != 1 {
			t.Fatalf("not original borrowed terminal: %#v", target)
		}
	}
	target, err := a.resolveSchema("#/components/schemas/S~0lot~1key", "/use")
	if err != nil || target.Catalog != catalog || target.Schema != original {
		t.Fatal("named catalog binding changed", target, err)
	}
	_, err = a.resolveSchema("#/components/schemas/S~0lot~1key/example/properties/leaf", "/use")
	if err == nil || err.Code != "E_REF_UNSUPPORTED" || *err.Pointer != "/use" {
		t.Fatal("opaque payload acquired schema role", err)
	}
}

func TestSchemaInternalScalarPointerHintTransferTypedTree(t *testing.T) {
	makeTree := func(check *PrimitiveValidation) TypeRef {
		ty := TypeRef{Kind: Scalar, Name: "kotlin.String", Validation: check}
		for _, kind := range []TypeKind{Nullable, List, Map, PatchValue, Named} {
			child := ty
			ty = TypeRef{Kind: kind, Name: "Alias", Element: &child}
		}
		return ty
	}
	for _, reverse := range []bool{false, true} {
		n := int64(1)
		left, right := &PrimitiveValidation{MinLength: &n}, &PrimitiveValidation{MinLength: &n}
		retained := &Model{ClassName: "Root", Nested: []Declaration{&Model{ClassName: "Child", Properties: []Property{{Name: "v", Type: makeTree(left)}}}}}
		candidate := &Model{ClassName: "Root", Nested: []Declaration{&Model{ClassName: "Child", Properties: []Property{{Name: "v", Type: makeTree(right)}}}}}
		// Exact semantic equality is the mandatory production caller precondition.
		if !reflect.DeepEqual(retained, candidate) {
			t.Fatal("test plans not semantically equal")
		}
		hints := EmissionHints{}
		if reverse {
			hints[left] = true
		} else {
			hints[right] = true
		}
		hints.transferEquivalent(retained, candidate)
		if !hints[left] || hints[nil] {
			t.Fatal("lost nested Nullable/Patch/list/map/transparent alias OR hint", reverse, hints)
		}
		if !reflect.DeepEqual(retained, candidate) {
			t.Fatal("hint transfer mutated semantic declarations")
		}
	}
}
