package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleScalarConstOriginalSourceAndActualCodec(t *testing.T) {
	root := loadComponentDocument(t, `{"Only":{"const":"fixed"},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"const":"fixed"}],"default":["fixed"]}}}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.consttuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	outer := holder.Properties[0].Type.Element
	leaf := outer.Element
	copyType := *leaf
	outer.Element = &copyType
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied const element retained original tuple authority")
	}
	outer.Element = leaf
	leaf.Name = "kotlin.Int"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed actual null codec retained tuple proof")
	}
	leaf.Name = "kotlin.String"
	var alias *ScalarAlias
	for _, d := range ds {
		if d.Name() == "OnlyDto" {
			alias = d.(*ScalarAlias)
		}
	}
	alias.Scalar = "kotlin.Int"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed actual null root declaration retained tuple authority")
	}
	alias.Scalar = "kotlin.String"
	only := a.components["Only"]
	scalar := only.Scalar
	only.Scalar = "kotlin.Int"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed original root backing retained tuple authority")
	}
	only.Scalar = scalar
	member := get(get(get(a.components["Holder"].Schema, "properties"), "rows"), "prefixItems").Items()[0]
	values := member
	replacementRoot, failure := document.Load([]byte(`{"const":"fixed"}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	replacement := get(replacementRoot, "const")
	original := values.Members()[0].Value
	values.Members()[0].Value = replacement
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte const literal replacement acquired original proof")
	}
	values.Members()[0].Value = original
	sources := a.tupleLiterals.Sources
	a.tupleLiterals.Sources = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("missing original const source proof was accepted")
	}
	a.tupleLiterals.Sources = sources
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
}
