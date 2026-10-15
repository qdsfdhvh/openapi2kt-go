package schema

import (
	"slices"
	"testing"

	"openapi-kt/internal/document"
)

func unionDefaultTestAnalyzer() *analyzer {
	return &analyzer{options: Options{Validation: true}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
}

func loadUnionField(t *testing.T, text string) field {
	t.Helper()
	value, err := document.Load([]byte(text), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	return field{Wire: "v", Pointer: "/v", Schema: value, Depth: 1}
}

func TestUnionFieldDefaultCandidateAndDeclaration(t *testing.T) {
	for _, tc := range []string{
		`{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"},{"type":"number","format":"double"},{"type":"boolean"}],"default":"x"}`,
		`{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"number"},{"type":"boolean"}],"default":3}`,
		`{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"number"},{"type":"boolean"}],"default":1.5}`,
		`{"oneOf":[{"type":"string"},{"type":"integer"},{"type":"number"},{"type":"boolean"}],"default":true}`,
	} {
		f := loadUnionField(t, tc)
		a := unionDefaultTestAnalyzer()
		if !a.primitiveUnionDefaultCandidate(f.Schema) {
			t.Fatal("candidate rejected", tc)
		}
		var nested []Declaration
		ty, err := a.unionDefaultField(f, "V", &nested, map[string]bool{})
		if err != nil || ty.Kind != Named || len(nested) != 1 {
			t.Fatal(ty, nested, err)
		}
		union, ok := nested[0].(*PrimitiveUnion)
		if !ok || union.Object != nil || union.NullVariant {
			t.Fatal(nested[0])
		}
		encoded, err := a.primitiveUnionDefaultEncoded(f, union)
		if err != nil || encoded == "" {
			t.Fatal(encoded, err)
		}
	}
}

func TestUnionFieldDefaultSelectorMirror(t *testing.T) {
	a := unionDefaultTestAnalyzer()
	// An integral token matches the Integer kind before Number, and its original
	// token is preserved for the runtime selector.
	integral := loadUnionField(t, `{"oneOf":[{"type":"integer"},{"type":"number"}],"default":1}`)
	var nested []Declaration
	if _, err := a.unionDefaultField(integral, "V", &nested, map[string]bool{}); err != nil {
		t.Fatal(err)
	}
	union := nested[0].(*PrimitiveUnion)
	if !slices.Contains(union.Kinds, PrimitiveInteger) || !slices.Contains(union.Kinds, PrimitiveNumber) {
		t.Fatal(union.Kinds)
	}
	encoded, err := a.primitiveUnionDefaultValue(integral, union)
	if err != nil || encoded != "1" {
		t.Fatal(encoded, err)
	}
	floatToken := loadUnionField(t, `{"oneOf":[{"type":"integer"},{"type":"number"}],"default":1.0}`)
	a2 := unionDefaultTestAnalyzer()
	var floatNested []Declaration
	floatType, err := a2.unionDefaultField(floatToken, "V", &floatNested, map[string]bool{})
	if err != nil {
		t.Fatal(floatType, err)
	}
	floatUnion := floatNested[0].(*PrimitiveUnion)
	encoded, err = a2.primitiveUnionDefaultValue(floatToken, floatUnion)
	if err != nil || encoded != "1.0" {
		t.Fatal("float token must be preserved", encoded, err)
	}
	// A token that matches no accepted kind is rejected.
	boolOnly := loadUnionField(t, `{"oneOf":[{"type":"boolean"}],"default":1}`)
	if _, err := a.unionDefaultField(boolOnly, "V", &[]Declaration{}, map[string]bool{}); err == nil {
		t.Fatal("integral token admitted for boolean-only union")
	}
}

func TestUnionFieldDefaultExclusions(t *testing.T) {
	for _, text := range []string{
		`{"oneOf":[{"type":"string","format":"date"},{"type":"integer"}],"default":"2024-01-02"}`,
		`{"oneOf":[{"type":"object","properties":{"a":{"type":"string"}}},{"type":"integer"}],"default":{"a":"x"}}`,
		`{"oneOf":[{"type":"string","nullable":true},{"type":"integer"}],"default":"x"}`,
	} {
		a := unionDefaultTestAnalyzer()
		if a.primitiveUnionDefaultCandidate(loadUnionField(t, text).Schema) {
			t.Fatal("excluded union shape admitted", text)
		}
	}
}

func TestUnionCollectionElementDefaultAuthority(t *testing.T) {
	field := loadUnionField(t, `{"type":"array","items":{"oneOf":[{"type":"string"},{"type":"integer","format":"int32"}]},"default":["x",1]}`)
	a := unionDefaultTestAnalyzer()
	if !a.objectDefaultCandidate(field.Schema) {
		t.Fatal("list-of-union not recognized by the collection gate")
	}
	node, err := a.objectDefaultNode(field.Schema, "/v", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if node.ty.Kind != List || node.element == nil || node.element.union == nil || node.element.schema != get(field.Schema, "items") {
		t.Fatal("element union node lost its declaration/source", node.element)
	}
	canonical, err := node.value(get(field.Schema, "default"), "/v/default")
	if err != nil || canonical != `["x",1]` {
		t.Fatal(canonical, err)
	}
	bad, err := document.Load([]byte(`{"v":["z",true]}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := node.value(get(bad, "v"), "/v/default"); err == nil || err.Pointer == nil || *err.Pointer != "/v/default/1" {
		t.Fatalf("element union member Pointer: %v", err)
	}
}
