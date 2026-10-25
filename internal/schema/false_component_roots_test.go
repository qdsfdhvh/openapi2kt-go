package schema

import (
	"bytes"
	"testing"

	"openapi-kt/internal/document"
)

func TestFalseComponentActualRootAndAliasAuthority(t *testing.T) {
	root, e := document.ParseObject([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Denied":false,"Alias":{"$ref":"#/components/schemas/Denied","nullable":true}}}}`), document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.falsevalues", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if a.validateInternalModels(ds) != nil {
		t.Fatal("valid false source/declarations rejected")
	}
	without := []Declaration{}
	for _, d := range ds {
		if _, ok := d.(*FalseValue); !ok {
			without = append(without, d)
		}
	}
	if a.validateInternalModels(without) == nil {
		t.Fatal("missing actual uninhabited declaration authorized")
	}
	var original *FalseValue
	var alias *ValueAlias
	for _, d := range ds {
		switch d := d.(type) {
		case *FalseValue:
			original = d
		case *ValueAlias:
			alias = d
		}
	}
	if original == nil || alias == nil {
		t.Fatal("missing originals")
	}
	if a.validateInternalModels(append(append([]Declaration{}, ds...), original)) == nil {
		t.Fatal("duplicate original false export admitted")
	}
	if a.validateInternalModels(append(without, &FalseValue{ClassName: original.ClassName})) == nil {
		t.Fatal("same-name copied false export admitted")
	}
	savedName := original.ClassName
	original.ClassName = "Changed"
	if a.validateInternalModels(ds) == nil {
		t.Fatal("changed false classifier admitted")
	}
	original.ClassName = savedName
	savedType := alias.Type
	alias.Type = TypeRef{Kind: JSON}
	if a.validateInternalModels(ds) == nil {
		t.Fatal("false alias raw codec replacement admitted")
	}
	alias.Type = savedType
	savedHeight := a.components["Alias"].AliasHeight
	a.components["Alias"].AliasHeight++
	if a.validateInternalModels(ds) == nil {
		t.Fatal("cached false alias work changed")
	}
	a.components["Alias"].AliasHeight = savedHeight
	before, _ := root.MarshalJSON()
	copyRoot, e := document.ParseObject(before, document.MaxDocumentBytes)
	if e != nil {
		t.Fatal(e)
	}
	savedSchema := a.components["Denied"].Schema
	a.components["Denied"].Schema = get(get(get(copyRoot, "components"), "schemas"), "Denied")
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte false source copied")
	}
	a.components["Denied"].Schema = savedSchema
	a.root = copyRoot
	if a.validateInternalModels(ds) == nil {
		t.Fatal("same-byte source ancestry copied")
	}
	a.root = root
	schema := a.components["Alias"].Schema
	copiedAlias := get(get(get(copyRoot, "components"), "schemas"), "Alias")
	for i, m := range schema.Members() {
		if m.Name == "nullable" {
			schema.Members()[i].Value = get(copiedAlias, "nullable")
			if a.validateInternalModels(ds) == nil {
				t.Fatal("same-byte nullable lookup/member divergence admitted")
			}
			schema.Members()[i].Value = m.Value
		}
	}
	if e := a.validateInternalModels(ds); e != nil {
		t.Fatal(e)
	}
	after, _ := root.MarshalJSON()
	if !bytes.Equal(before, after) {
		t.Fatal("original false source mutated")
	}

}
