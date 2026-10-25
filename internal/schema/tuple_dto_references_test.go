package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleDTOActualConstructorSourceAndAliasAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"properties":{"value":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Item","nullable":false},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"},{"type":"string"}],"items":{"type":"integer"}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	before, _ := root.MarshalJSON()
	member := get(a.components["Probe"].Schema, "prefixItems").Items()[0]
	work := 0
	signature, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work)
	if err != nil || signature.ModelSource != a.components["Item"].Schema || signature.Nullable || work != 2 || len(a.borrowedDTOLeaves) != 1 {
		t.Fatal("lost original constructor/nullable/source accounting", signature, work, err)
	}
	for _, depth := range []int{MaxSchemaDepth - 2, MaxSchemaDepth - 1} {
		work = 0
		_, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", depth, &work)
		if depth == MaxSchemaDepth-2 && err != nil || depth == MaxSchemaDepth-1 && (err == nil || err.Code != "E_SCHEMA_DEPTH") {
			t.Fatal("nominal barrier erased reference hops", depth, err)
		}
	}
	for _, spare := range []int{2, 1} {
		work = 16384 - spare
		_, err := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work)
		if spare == 2 && err != nil || spare == 1 && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("nominal barrier reset tuple work", work, err)
		}
	}
	// Reanalyze into a separate initialized analyzer to keep the existing source
	// identity maps intact while checking final declaration/alias authority.
	b := analyzer{root: root, options: a.options, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := b.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	var actual *Model
	without := []Declaration{}
	for _, d := range ds {
		if d.Name() == "ItemDto" {
			actual = d.(*Model)
		} else {
			without = append(without, d)
		}
	}
	if actual == nil {
		t.Fatal("missing actual constructor")
	}
	for _, invalid := range [][]Declaration{without, append(append([]Declaration{}, without...), &Model{ClassName: actual.ClassName, Properties: actual.Properties}), append(append([]Declaration{}, ds...), actual)} {
		if b.validateInternalModels(invalid) == nil {
			t.Fatal("missing/copied/ambiguous constructor authorized a discarded typed prefix")
		}
	}
	for _, name := range []string{"Item", "Alias"} {
		original := b.components[name].Schema
		bytes, _ := original.MarshalJSON()
		copy, err := document.Load(bytes, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		b.components[name].Schema = copy
		if b.validateInternalModels(ds) == nil {
			t.Fatal("same-byte source/alias replacement authorized", name)
		}
		b.components[name].Schema = original
	}
	if err := b.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("tuple manufactured source nodes")
	}
}

func TestTupleEnumDiscardedPrefixRetainsAliasSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"string","enum":["ready","done"]},"Alias":{"$ref":"#/components/schemas/Item","nullable":false},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"items":{"type":"integer"}}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.tupleenums", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	before := a.components["Alias"].Schema
	raw, _ := before.MarshalJSON()
	copy, err := document.Load(raw, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a.components["Alias"].Schema = copy
	if a.validateInternalModels(ds) == nil {
		t.Fatal("discarded enum prefix authorized a same-byte alias source copy")
	}
	a.components["Alias"].Schema = before
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}
