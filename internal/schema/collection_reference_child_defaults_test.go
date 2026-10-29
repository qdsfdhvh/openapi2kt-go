package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestCollectionReferenceChildDefaultSourceAndTypeAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Word":{"enum":["one","two"],"minLength":2},"Other":{"enum":["one","two"],"minLength":2},"Alias":{"$ref":"#/components/schemas/Word"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":"one"}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.childdefault", Suffix: "Dto", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("finalize", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	node := get(a.components["Rows"].Schema, "items")
	source := a.collectionChildDefaults[node]
	if source == nil || len(source.Uses) == 0 {
		t.Fatal("missing child source or use")
	}
	use := source.Uses[0]
	if use.Actual == nil || use.Actual.Validation == nil {
		t.Fatal("missing actual checked enum element")
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Fatal("changed child annotation retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored authority rejected", name, failure)
		}
	}
	reject("canonical value", func() func() {
		old := source.Encoded
		source.Encoded = `"two"`
		return func() { source.Encoded = old }
	})
	reject("raw value", func() func() {
		old := source.ValueBytes
		source.ValueBytes = `"two"`
		return func() { source.ValueBytes = old }
	})
	reject("depth", func() func() {
		use.Depth++
		return func() { use.Depth-- }
	})
	reject("preflight type", func() func() {
		old := use.Type.Validation
		use.Type.Validation = nil
		return func() { use.Type.Validation = old }
	})
	reject("actual type check", func() func() {
		old := use.Actual.Validation
		use.Actual.Validation = nil
		return func() { use.Actual.Validation = old }
	})
	reject("same-value actual type copy", func() func() {
		old := use.Actual
		copy := *old
		use.Actual = &copy
		return func() { use.Actual = old }
	})
	reject("actual container element replaced", func() func() {
		rows := componentDeclaration(declarations, "RowsDto").(*ValueAlias)
		old := rows.Type.Element
		if old == nil {
			t.Fatal("missing actual collection element")
		}
		copy := *old
		rows.Type.Element = &copy
		return func() { rows.Type.Element = old }
	})
	reject("alias rebound to a different enum", func() func() {
		alias := a.components["Alias"]
		old := alias.BindingTarget
		alias.BindingTarget = a.components["Other"]
		return func() { alias.BindingTarget = old }
	})
	reject("default node replaced", func() func() {
		members := node.Members()
		for i := range members {
			if members[i].Name == "default" {
				old := members[i].Value
				copy, failure := document.Load([]byte(`{"value":"one"}`), document.JSON, "")
				if failure != nil {
					t.Fatal(failure)
				}
				members[i].Value = get(copy, "value")
				return func() { members[i].Value = old }
			}
		}
		t.Fatal("missing default member")
		return nil
	})
	reject("use dropped", func() func() {
		old := source.Uses
		source.Uses = nil
		return func() { source.Uses = old }
	})
}
