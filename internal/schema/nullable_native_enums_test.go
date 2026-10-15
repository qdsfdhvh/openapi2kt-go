package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestNullableNativeEnumOwnsActualNullableExportAndCanonicalChecks(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Code":{"type":["integer","null"],"enum":[1,2],"minimum":2},"Alias":{"$ref":"#/components/schemas/Code","nullable":false},"Source":{"properties":{"hop":{"$ref":"#/components/schemas/Alias"}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"}}}}`)
	root, e := document.Load(input, document.JSON, "")
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.enums", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	c := a.components["Code"]
	origin := a.nativeEnumOrigins[c]
	actual, ok := origin.Actual.(*ValueAlias)
	if !ok || !c.NullableNativeEnum || actual.Type.Kind != Nullable || bare(actual.Type).Validation != c.Validation || a.componentType(a.components["Alias"]).Kind != Scalar || a.componentType(a.components["Bridge"]).Kind != Scalar {
		t.Fatal(origin)
	}
	if err := a.validateNativeEnumBindings(ds); err != nil {
		t.Fatal(err)
	}
	prior := actual.Type
	actual.Type = actual.Type.WithoutRootNull()
	if err := a.validateNativeEnumBindings(ds); err == nil {
		t.Fatal("narrowed physical nullable export authorized")
	}
	actual.Type = prior
	copy := *prior.Element
	copyValidation := *c.Validation
	copy.Validation = &copyValidation
	actual.Type.Element = &copy
	if err := a.validateNativeEnumBindings(ds); err == nil {
		t.Fatal("same-check replacement ValueAlias plan authorized")
	}
	actual.Type = prior
	if err := a.validateNativeEnumBindings(ds); err != nil {
		t.Fatal(err)
	}
	bytes, _ := root.MarshalJSON()
	if string(bytes) != string(input) {
		t.Fatal("source rewritten")
	}
}
