package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestTupleBooleanReferenceOriginalSourceAndActualTypes(t *testing.T) {
	root := loadComponentDocument(t, `{"Empty":false,"Maybe":{"$ref":"#/components/schemas/Empty","nullable":true,"default":null},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Maybe"}],"default":[null]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Empty"}],"default":[]}}}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.boolrefs", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	plan := &Plan{Models: ds}
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	holder := componentModel(ds, "HolderDto")
	element := holder.Properties[0].Type.Element.Element
	original := *element
	element.Name = "org.example.boolrefs.models.OtherDto"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("altered actual tuple reference element kept authority")
	}
	*element = original
	copyType := *element
	outer := holder.Properties[0].Type.Element
	outer.Element = &copyType
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("same-byte copied element acquired actual source authority")
	}
	outer.Element = element
	var classifier *FalseValue
	for _, d := range ds {
		if d, ok := d.(*FalseValue); ok {
			classifier = d
		}
	}
	if classifier == nil {
		t.Fatal("missing original false root")
	}
	name := classifier.ClassName
	classifier.ClassName = "OtherDto"
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("renamed root classifier kept tuple reference proof")
	}
	classifier.ClassName = name
	for i, d := range ds {
		if d != classifier {
			continue
		}
		copyRoot := *classifier
		ds[i] = &copyRoot
		if a.validateTuplePlan(plan) == nil {
			t.Fatal("copied root classifier kept original source authority")
		}
		ds[i] = classifier
	}
	maybe := a.components["Maybe"]
	value, failure := document.Load([]byte(`{"$ref":"#/components/schemas/Empty","nullable":true,"default":null}`), document.JSON, "")
	if failure != nil {
		t.Fatal(failure)
	}
	old := maybe.Schema
	maybe.Schema = value
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("copied nullable alias acquired tuple source authority")
	}
	maybe.Schema = old
	uses := a.tupleBooleanRefs.Uses
	a.tupleBooleanRefs.Uses = map[*TypeRef]tupleBooleanReferenceUse{}
	if a.validateTuplePlan(plan) == nil {
		t.Fatal("missing retained selected element proof was accepted")
	}
	a.tupleBooleanRefs.Uses = uses
	if err := a.validateTuplePlan(plan); err != nil {
		t.Fatal(err)
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("Boolean reference planning mutated source")
	}
}
