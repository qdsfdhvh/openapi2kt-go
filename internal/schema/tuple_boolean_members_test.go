package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleBooleanRetainedHelperAndOriginalSource(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","prefixItems":[false]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[false],"default":[]}}}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.booleanmembers", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	helper := holder.Nested[0].(*FalseValue)
	holder.Nested = nil
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("missing false helper retained source authority")
	}
	copy := *helper
	holder.Nested = []Declaration{&copy}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-name helper copy retained source authority")
	}
	holder.Nested = []Declaration{helper, helper}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("duplicate helper retained authority")
	}
	holder.Nested = []Declaration{helper}
	plan.Models = append(append([]Declaration{}, ds...), helper)
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same helper exported in two scopes retained unique declaration authority")
	}
	plan.Models = ds
	old := helper.ClassName
	helper.ClassName = "Other"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("renamed helper retained old type proof")
	}
	helper.ClassName = old
	outer := holder.Properties[0].Type.Element
	value := outer.Element
	copyType := *value
	holder.Properties[0].Type.Element = &TypeRef{Kind: List, Element: &copyType}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied element address acquired actual source proof")
	}
	holder.Properties[0].Type.Element = outer
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	prefix := get(a.components["Rows"].Schema, "prefixItems")
	original := prefix.Items()[0]
	copyValue, failure := document.Load([]byte(`{"value":false}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	prefix.Items()[0] = get(copyValue, "value")
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte Boolean source copy retained authority")
	}
	prefix.Items()[0] = original
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("Boolean tuple planning rewrote sources")
	}
}
