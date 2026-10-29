package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestStringEnumCollectionBoundsRetainOriginalLeafAndChecks(t *testing.T) {
	input := []byte(`{"type":["string","null"],"enum":[null,"01","true","x"],"minLength":2,"maxLength":4,"pattern":"[0-9]|true"}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{options: Options{Validation: true}, validationSources: map[*PrimitiveValidation]string{}, emissionHints: EmissionHints{}, enumOrigins: map[Declaration]nominalOrigin{}}
	var nested []Declaration
	ty, failure := a.stringEnumCollectionLeaf(root, "CodesInner", "/original/field/items", &nested, map[string]bool{}, 2)
	if failure != nil || ty.Kind != Nullable || len(nested) != 1 {
		t.Fatal(ty, failure)
	}
	leaf := *ty.Element
	enum := nested[0].(*StringEnum)
	checks := leaf.Validation
	origin := a.enumOrigins[enum]
	if leaf.Kind != Named || leaf.Name != "CodesInner" || leaf.Element != nil || checks == nil || len(checks.Enum) != 0 || a.validationSources[checks] != "/original/field/items" || !a.emissionHints[checks] || origin.Schema != root || origin.Pointer != "/original/field/items" || !origin.Nullable {
		t.Fatal(leaf, checks, origin)
	}
	if enum.Members[0].Wire != "01" || enum.Members[1].Wire != "true" || enum.Members[2].Wire != "x" {
		t.Fatal("Vocabulary rewritten", enum)
	}
	after, marshalErr := root.MarshalJSON()
	if marshalErr != nil || string(after) != string(input) {
		t.Fatal("Source mutated", marshalErr)
	}
}

func TestStringEnumFieldBoundsRetainOriginalSourceAndChecks(t *testing.T) {
	input := []byte(`{"type":["string","null"],"format":"date","enum":["one","x"],"minLength":2,"maxLength":4,"default":"one"}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{options: Options{Validation: true}, validationSources: map[*PrimitiveValidation]string{}, emissionHints: EmissionHints{}, enumOrigins: map[Declaration]nominalOrigin{}}
	var ds []Declaration
	ty, failure := a.checkedStringEnumLeaf(root, "State", "/original/properties/state", &ds, map[string]bool{}, 1, true)
	if failure != nil || ty.Kind != Nullable || len(ds) != 1 {
		t.Fatal("missing original nullable bounded enum field", ty, failure)
	}
	leaf := *ty.Element
	origin := a.enumOrigins[ds[0]]
	if leaf.Kind != Named || leaf.Element != nil || leaf.Validation == nil || len(leaf.Validation.Enum) != 0 || a.validationSources[leaf.Validation] != origin.Pointer || origin.Schema != root || origin.Pointer != "/original/properties/state" || !a.emissionHints[leaf.Validation] {
		t.Fatal("bounded lexical field lost source/check authority", leaf, origin)
	}
	after, marshalErr := root.MarshalJSON()
	if marshalErr != nil || string(after) != string(input) {
		t.Fatal("bounded lexical field rewrote source vocabulary/default/metadata", marshalErr)
	}
	// Default ownership belongs to the direct field; collection children keep
	// their existing own-default exclusion on the very same original source.
	if _, err := a.checkedStringEnumLeaf(root, "Child", "/original/items", nil, nil, 1, false); err == nil {
		t.Fatal("field annotation role leaked into a collection child")
	}
}
