package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestBorrowedCollectionRawTrueLeavesActualSourceAuthority(t *testing.T) {
	for _, terminal := range []string{`{"type":"object","nullable":true}`, `{"allOf":[{"type":"object"},{"properties":{}}]}`, `true`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"}}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/rows","nullable":false}}}}`)
		root, err := document.Load(input, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.rawtrue", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(terminal, err)
		}
		c := a.components["Value"]
		if len(a.borrowedRawTrueLeaves) != 1 {
			t.Fatal("original raw/true source not retained")
		}
		missing := []Declaration{}
		var actual Declaration
		for _, d := range ds {
			if d.Name() == c.Name {
				actual = d
			} else {
				missing = append(missing, d)
			}
		}
		if err := a.validateBorrowedRawTrueLeaves(missing); err == nil {
			t.Fatal("missing actual source authorized")
		}
		var copy Declaration
		if terminal == `true` {
			copy = &ValueAlias{ClassName: c.Name, Type: TypeRef{Kind: JSON}}
		} else {
			copy = &Model{ClassName: c.Name, Nullable: terminal == `{"type":"object","nullable":true}`}
		}
		if err := a.validateBorrowedRawTrueLeaves(append(append([]Declaration{}, missing...), copy)); err == nil {
			t.Fatal("same-name source replacement authorized")
		}
		a.rawTrueOrigins[copy] = a.rawTrueOrigins[actual]
		if m, ok := copy.(*Model); ok {
			a.modelOrigins[m] = c.Schema
			a.modelPointers[m] = c.Pointer
		}
		if err := a.validateBorrowedRawTrueLeaves(append(append([]Declaration{}, ds...), copy)); err == nil {
			t.Fatal("multiple actual sources authorized")
		}
		delete(a.rawTrueOrigins, copy)
		if err := a.validateBorrowedRawTrueLeaves(append(append([]Declaration{}, ds...), actual)); err == nil {
			t.Fatal("repeat original source occurrence authorized")
		}
		saved := c.Schema
		copiedDocument, err := document.Load([]byte(`{"value":`+terminal+`}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		c.Schema = get(copiedDocument, "value")
		if err := a.validateBorrowedRawTrueLeaves(ds); err == nil {
			t.Fatal("samebytes copied source authorized")
		}
		c.Schema = saved
		if alias, ok := actual.(*ValueAlias); ok {
			alias.Type.Serializer = "synthetic.Override"
			if err := a.validateBorrowedRawTrueLeaves(ds); err == nil {
				t.Fatal("changed literal true codec authorized")
			}
			alias.Type.Serializer = ""
		}
		if m, ok := actual.(*Model); ok {
			m.Nullable = !m.Nullable
			if err := a.validateBorrowedRawTrueLeaves(ds); err == nil {
				t.Fatal("changed raw Model nullable backing authorized")
			}
			m.Nullable = !m.Nullable
			m.Constants = append(m.Constants, Constant{Name: "kind", WireName: "kind", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}, Initializer: `"constant"`})
			if err := a.validateBorrowedRawTrueLeaves(ds); err == nil {
				t.Fatal("const-only Model granted raw source role")
			}
			m.Constants = nil
		}
		if err := a.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("original raw/true source mutated")
		}
	}
}
