package kotlin

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/internal/schema"
)

func TestNullableArrayLazyViewRenderingAndChecks(t *testing.T) {
	var backing schema.TypeRef
	view := schema.TypeRef{Kind: schema.Named, Name: "org.example.models.Rows", Element: &backing, ArrayAliasView: schema.NullableArrayAlias}
	narrow := view.Null().WithoutRootNull()
	// Narrowing before the forward plan is bound must not copy its zero value.
	if narrow.Element != &backing || narrow.Kind != schema.Named || narrow.ArrayAliasView != schema.NonNullArrayAlias {
		t.Fatal(narrow)
	}
	leaf := schema.TypeRef{Kind: schema.Named, Name: "RowsInner"}
	item := leaf.Null()
	mapValue := schema.TypeRef{Kind: schema.Map, Element: &item}
	n := int64(1)
	list := schema.TypeRef{Kind: schema.List, Element: &mapValue, Collection: &schema.CollectionValidation{Minimum: &n, MinimumKey: "minItems"}}
	backing = list.Null()
	before := backing
	if got := RenderType(narrow); got != "kotlin.collections.List<kotlin.collections.Map<kotlin.String, org.example.models.RowsInner?>>" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(backing, before) || RenderType(view.Null()) != "org.example.models.Rows?" {
		t.Fatal("render mutated shared nullable alias", backing)
	}
	var out strings.Builder
	emitTypeChecks(narrow, "input", "Field value", 0, &out)
	if !strings.Contains(out.String(), "require(value.size >= 1)") || strings.Contains(out.String(), "input?.let") {
		t.Fatal("narrow view must enforce root checks without nullable skip", out.String())
	}
	out.Reset()
	emitTypeChecks(view.Null(), "input", "Field value", 0, &out)
	if !strings.Contains(out.String(), "input?.let") {
		t.Fatal(out.String())
	}
	if !reflect.DeepEqual(backing, before) {
		t.Fatal("checks mutated shared target")
	}
}
