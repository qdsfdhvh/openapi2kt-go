package schema

import (
	"openapi-kt/internal/document"
	"testing"
)

func TestReferencedScalarFieldDefaultsRetainSourceChecksAndAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/v","default":2}}},"Hops":{"properties":{"v":{"$ref":"#/components/schemas/Alias","nullable":false}}},"Alias":{"$ref":"#/components/schemas/Value"},"Value":{"type":"integer","format":"int64","nullable":true,"minimum":0}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	canonical := a.components["Value"]
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	plan := pending[0]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if err != nil || node.schema != canonical.Schema || node.pointer != canonical.Pointer || node.ty.Validation != canonical.Validation || node.ty.Kind != Scalar || node.ty.Name != "kotlin.Long" || a.validationSources[node.ty.Validation] != canonical.Pointer {
		t.Fatal("source/check/nearest false drift", err, node)
	}
	if plan.Model.Properties[plan.Index].Default != "2L" {
		t.Fatal("wrong actual literal")
	}
	missing := []Declaration{}
	var original Declaration
	for _, d := range ds {
		if d.Name() == canonical.Name {
			original = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateBuiltinBindings(missing); err == nil {
		t.Fatal("missing declaration authorized")
	}
	duplicate := &ValueAlias{ClassName: canonical.Name, Type: a.componentType(canonical)}
	a.builtinOrigins[duplicate] = a.builtinOrigins[original]
	if err := a.validateBuiltinBindings(append(append([]Declaration{}, ds...), duplicate)); err == nil {
		t.Fatal("duplicate declaration authorized")
	}
	delete(a.builtinOrigins, duplicate)
	saved := canonical.Validation
	copy := *saved
	canonical.Validation = &copy
	if err := a.validateBuiltinBindings(ds); err == nil {
		t.Fatal("replaced check authorized")
	}
	canonical.Validation = saved
	if err := a.validateBuiltinBindings(ds); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
}
