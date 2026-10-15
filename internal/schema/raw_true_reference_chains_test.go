package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestRawTrueReferenceChainsActualSourceAuthorityAndCache(t *testing.T) {
	for _, terminal := range []string{`{"type":"object","nullable":true}`, `{"allOf":[{"type":"object"},{"properties":{}}]}`, `true`} {
		input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Value":` + terminal + `,"Alias":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Alias"},"false":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false},"again":{"$ref":"#/components/schemas/Bridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/false","nullable":true},"Again":{"$ref":"#/components/schemas/Hops/properties/again","nullable":false}}}}`)
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
		for _, tc := range []struct {
			path     string
			steps    int
			nullable bool
		}{{"Hops/properties/value", 3, true}, {"Hops/properties/false", 4, false}, {"Hops/properties/again", 6, true}} {
			target, err := a.resolveSchema("#/components/schemas/"+tc.path, "/use")
			if err != nil {
				t.Fatal(err)
			}
			if target.CanonicalJSON != c || target.Schema != c.Schema || target.Pointer != c.Pointer || target.Steps != tc.steps || target.ModelName != "" || target.Deferred != nil || len(a.components) != 5 {
				t.Fatal("JSON role gained fake nominal/source", tc, target)
			}
			if flag, err := target.effectiveNullable(); err != nil || flag != tc.nullable {
				t.Fatal(tc, flag, err)
			}
			a.aliasPlanning = true
			no := false
			ty, err := a.referenceType(target, &no, 48-tc.steps, "/depth48")
			if err != nil || ty.Kind != JSON || ty.Slots() != 1 || ty.SchemaSteps != tc.steps {
				t.Fatal(ty, err)
			}
			if _, err := a.referenceType(target, &no, 49-tc.steps, "/depth49"); err == nil {
				t.Fatal("JSONfalse reset source height")
			}
			a.aliasPlanning = false
		}
		if a.componentType(a.components["Bridge"]).Kind != Nullable || a.componentType(a.components["Again"]).Kind != JSON {
			t.Fatal("nearest root override lost")
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
		if err := a.validateRawTrueBindings(missing); err == nil {
			t.Fatal("missing actual source authorized")
		}
		var copy Declaration
		if terminal == `true` {
			copy = &ValueAlias{ClassName: c.Name, Type: TypeRef{Kind: JSON}}
		} else {
			copy = &Model{ClassName: c.Name, Nullable: terminal == `{"type":"object","nullable":true}`}
		}
		if err := a.validateRawTrueBindings(append(append([]Declaration{}, missing...), copy)); err == nil {
			t.Fatal("same-name source replacement authorized")
		}
		a.rawTrueOrigins[copy] = a.rawTrueOrigins[actual]
		if m, ok := copy.(*Model); ok {
			a.modelOrigins[m] = c.Schema
			a.modelPointers[m] = c.Pointer
		}
		if err := a.validateRawTrueBindings(append(append([]Declaration{}, ds...), copy)); err == nil {
			t.Fatal("multiple actual sources authorized")
		}
		delete(a.rawTrueOrigins, copy)
		if err := a.validateRawTrueBindings(append(append([]Declaration{}, ds...), actual)); err == nil {
			t.Fatal("repeat original source occurrence authorized")
		}
		saved := c.Schema
		copiedDocument, err := document.Load([]byte(`{"value":`+terminal+`}`), document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		c.Schema = get(copiedDocument, "value")
		if err := a.validateRawTrueBindings(ds); err == nil {
			t.Fatal("samebytes copied source authorized")
		}
		c.Schema = saved
		if alias, ok := actual.(*ValueAlias); ok {
			alias.Type.Serializer = "synthetic.Override"
			if err := a.validateRawTrueBindings(ds); err == nil {
				t.Fatal("changed literal true codec authorized")
			}
			alias.Type.Serializer = ""
		}
		if m, ok := actual.(*Model); ok {
			m.Nullable = !m.Nullable
			if err := a.validateRawTrueBindings(ds); err == nil {
				t.Fatal("changed raw Model nullable backing authorized")
			}
			m.Nullable = !m.Nullable
			m.Constants = append(m.Constants, Constant{Name: "kind", WireName: "kind", Type: TypeRef{Kind: Scalar, Name: "kotlin.String"}, Initializer: `"constant"`})
			if err := a.validateRawTrueBindings(ds); err == nil {
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
