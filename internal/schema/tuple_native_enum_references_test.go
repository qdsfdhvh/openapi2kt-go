package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleNativeEnumDiscardedInternalOriginalSource(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"code":{"type":"integer","enum":[1,2]}}},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Source/properties/code"}],"items":{"type":"string"}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.tuplenative", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	terminal := get(get(a.components["Source"].Schema, "properties"), "code")
	vocabulary := get(terminal, "enum")
	copy, e := document.Load([]byte(`{"one":1}`), document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	old := vocabulary.Items()[0]
	vocabulary.Items()[0] = get(copy, "one")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("discarded native tuple accepted same-byte original member replacement")
	}
	vocabulary.Items()[0] = old
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
}

func TestTupleNativeEnumNamedExportAndAliasAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":"integer","enum":[1,2]},"Alias":{"$ref":"#/components/schemas/Code","nullable":true},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}],"items":{"type":"string"}}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.tuplenative", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	c := a.components["Code"]
	origin := a.nativeEnumOrigins[c]
	if origin == nil {
		t.Fatal("missing original native export")
	}
	missing := []Declaration{}
	for _, d := range ds {
		if d != origin.Actual {
			missing = append(missing, d)
		}
	}
	if a.validateInternalModels(missing) == nil {
		t.Fatal("discarded prefix lost original export authority")
	}
	if a.validateInternalModels(append(append([]Declaration{}, ds...), origin.Actual)) == nil {
		t.Fatal("duplicate native export admitted")
	}
	copied := &ScalarAlias{ClassName: origin.Actual.Name(), Scalar: origin.Scalar}
	if a.validateInternalModels(append(missing, copied)) == nil {
		t.Fatal("copied native export admitted")
	}
	old := c.Validation.Enum[0]
	c.Validation.Enum[0] = "99"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("mutated membership admitted")
	}
	c.Validation.Enum[0] = old
	saved := a.components["Alias"]
	copyAlias := *saved
	a.components["Alias"] = &copyAlias
	if a.validateInternalModels(ds) == nil {
		t.Fatal("copied alias catalog admitted")
	}
	a.components["Alias"] = saved
	copyRoot, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a.root = copyRoot
	if a.validateInternalModels(ds) == nil {
		t.Fatal("copied original root admitted")
	}
	a.root = root
	work := 16383
	member := get(a.components["Probe"].Schema, "prefixItems").Items()[0]
	if _, e := a.tupleScalar(member, "/components/schemas/Probe/prefixItems/0", 1, &work); e == nil || e.Code != "E_MODEL_LIMIT" {
		t.Fatal("erased alias source work", e)
	}
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
}
