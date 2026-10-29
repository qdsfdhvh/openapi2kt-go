package schema

import "testing"

func TestTupleMetadataEnumDeclarationAuthority(t *testing.T) {
	for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
		root := loadComponentDocument(t, `{"Value":{"type":"string","enum":["ready"],"format":"date"},"Alias":{"$ref":"#/components/schemas/Value"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Alias"}]`+tail+`}}`)
		a := analyzer{root: root, options: Options{PackageName: "org.example.metadata", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(tail, err)
		}
		plan := &Plan{Models: ds}
		if err := a.validateTuplePlan(plan); err != nil {
			t.Fatal(tail, err)
		}
		var actual *StringEnum
		index := -1
		for i, d := range ds {
			if d.Name() == "ValueDto" {
				actual = d.(*StringEnum)
				index = i
			}
		}
		if actual == nil {
			t.Fatal("missing original lexical enum")
		}
		member := actual.Members[0]
		actual.Members[0].Wire = "changed"
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("changed retained vocabulary kept authority", tail)
		}
		actual.Members[0] = member
		copy := *actual
		ds[index] = &copy
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("copied lexical declaration acquired authority", tail)
		}
		ds[index] = actual
		if a.validateTuplePlan(&Plan{Models: append(append([]Declaration{}, ds...), actual)}) == nil {
			t.Fatal("duplicate lexical declaration kept authority", tail)
		}
		alias := a.components["Alias"]
		alias.AliasHeight++
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("erased cached height changed without rejection", tail)
		}
		alias.AliasHeight--
		if err := a.validateTuplePlan(plan); err != nil {
			t.Fatal(err)
		}
	}
}
