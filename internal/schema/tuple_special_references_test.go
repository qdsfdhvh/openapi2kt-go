package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleSpecialOriginalCodecAndTypeAuthority(t *testing.T) {
	for _, schema := range []string{`{"x-native-type":"bigint"}`, `{"type":"string","format":"byte"}`, `{"type":"string","x-kotlin-value-class":true}`, `{"x-native-type":"org.example.provider.Money"}`, `{"x-native-type":"date"}`, `{"type":"object","properties":{}}`, `{"type":"object","additionalProperties":true}`} {
		root := loadComponentDocument(t, `{"Value":`+schema+`,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Maybe"}]}}`)
		before, _ := root.MarshalJSON()
		a := analyzer{root: root, options: Options{PackageName: "org.example.special", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
		ds, err := a.analyze(root)
		if err != nil {
			t.Fatal(schema, err)
		}
		plan := &Plan{Models: ds}
		if err := a.validateTuplePlan(plan); err != nil {
			t.Fatal(schema, err)
		}
		var rows *ValueAlias
		var actual Declaration
		for _, d := range ds {
			if d.Name() == "RowsDto" {
				rows = d.(*ValueAlias)
			} else if d.Name() == "ValueDto" {
				actual = d
			}
		}
		if rows == nil || actual == nil {
			t.Fatal("missing actual source/consumer", schema)
		}
		element := rows.Type.Element
		copyType := *element
		rows.Type.Element = &copyType
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("same-byte copied consuming element acquired authority", schema)
		}
		rows.Type.Element = element
		duplicate := append(append([]Declaration{}, ds...), actual)
		if a.validateTuplePlan(&Plan{Models: duplicate}) == nil {
			t.Fatal("duplicated actual source export kept authority", schema)
		}
		if len(a.tupleSpecials.Order) != 0 {
			source := a.tupleSpecials.Order[0]
			source.Steps++
			if a.validateTuplePlan(plan) == nil {
				t.Fatal("mutated erased source height kept authority", schema)
			}
			source.Steps--
		}
		c := a.components["Maybe"]
		c.AliasHeight++
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("mutated cached alias height kept authority", schema)
		}
		c.AliasHeight--
		// Replace an original terminal member with a same-byte node. Neither
		// parser lookup nor a matching output spelling grants source authority.
		terminal := a.components["Value"].Schema
		member := terminal.Members()[0]
		data, _ := member.Value.MarshalJSON()
		copyRoot, failure := document.Load(append(append([]byte(`{"value":`), data...), '}'), document.JSON, "")
		if failure != nil {
			t.Fatal(failure)
		}
		terminal.Members()[0].Value = get(copyRoot, "value")
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("same-byte scalar/marker source acquired authority", schema)
		}
		terminal.Members()[0].Value = member.Value
		if err := a.validateTuplePlan(plan); err != nil {
			t.Fatal(schema, err)
		}
		after, _ := root.MarshalJSON()
		if string(before) != string(after) {
			t.Fatal("tuple special planning mutated source", schema)
		}
	}
}
