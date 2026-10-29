package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleNullOriginalSourceAndActualCodec(t *testing.T) {
	root := loadComponentDocument(t, `{"Only":{"type":"null"},"Alias":{"$ref":"#/components/schemas/Only"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"enum":[null]}],"default":[null]}}}}`)
	a := analyzer{root: root, options: Options{PackageName: "org.example.nulltuples", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
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
		t.Fatal("copied JsonNull element retained original tuple authority")
	}
	outer.Element = leaf
	leaf.Name = "kotlin.String"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed actual null codec retained tuple proof")
	}
	leaf.Name = "kotlinx.serialization.json.JsonNull"
	var alias *ScalarAlias
	for _, d := range ds {
		if d.Name() == "OnlyDto" {
			alias = d.(*ScalarAlias)
		}
	}
	alias.Scalar = "kotlin.String"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed actual null root declaration retained tuple authority")
	}
	alias.Scalar = "kotlinx.serialization.json.JsonNull"
	only := a.components["Only"]
	scalar := only.Scalar
	only.Scalar = "kotlin.String"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("changed original root backing retained tuple authority")
	}
	only.Scalar = scalar
	member := get(get(get(a.components["Holder"].Schema, "properties"), "rows"), "prefixItems").Items()[0]
	values := get(member, "enum")
	replacementRoot, failure := document.Load([]byte(`{"null":null}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	replacement := get(replacementRoot, "null")
	original := values.Items()[0]
	values.Items()[0] = replacement
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte null vocabulary replacement acquired original proof")
	}
	values.Items()[0] = original
	sources := a.tupleLiterals.Sources
	a.tupleLiterals.Sources = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("missing original JsonNull source proof was accepted")
	}
	a.tupleLiterals.Sources = sources
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
}
