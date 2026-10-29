package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestInlineEnumCollectionChildDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Rows":{"type":"array","items":{"additionalProperties":{"enum":["one","two"],"minLength":2,"default":"one"}}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:               Options{PackageName: "org.example.childdefault", Suffix: "Dto", Validation: true},
		components:            map[string]*component{},
		emissionHints:         EmissionHints{},
		validationSources:     map[*PrimitiveValidation]string{},
		collectionSources:     map[*CollectionValidation]string{},
		aliasUses:             map[collectionAliasUse]struct{}{},
		enumDiagnosticOrigins: map[*StringEnum]nominalOrigin{},
	}
	declarations, failure := a.analyze(root)
	if failure != nil {
		t.Fatal("analyze", failure)
	}
	if failure = a.finalizeInlineEnumCollectionChildDefaults(declarations, nil); failure != nil {
		t.Fatal("finalize", failure)
	}
	if failure = a.validateInternalModels(declarations); failure != nil {
		t.Fatal("original authority", failure)
	}
	node := get(get(a.components["Rows"].Schema, "items"), "additionalProperties")
	source := a.inlineEnumChildDefaults[node]
	if source == nil {
		t.Fatal("missing source")
	}
	var use *inlineEnumCollectionChildUse
	for _, candidate := range source.Uses {
		if candidate.Retained {
			use = candidate
			break
		}
	}
	if use == nil || use.Enum == nil || use.Actual.WithoutRootNull().Validation == nil {
		t.Fatal("missing retained checked enum element")
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInternalModels(declarations); failure == nil {
			t.Fatal("changed source or type retained authority", name)
		}
		restore()
		if failure := a.validateInternalModels(declarations); failure != nil {
			t.Fatal("restored authority rejected", name, failure)
		}
	}
	reject("default node replaced", func() func() {
		for i := range node.Members() {
			if node.Members()[i].Name == "default" {
				old := node.Members()[i].Value
				copy, err := document.Load([]byte(`{"v":"one"}`), document.JSON, "")
				if err != nil {
					t.Fatal(err)
				}
				node.Members()[i].Value = get(copy, "v")
				return func() { node.Members()[i].Value = old }
			}
		}
		t.Fatal("missing default")
		return nil
	})
	reject("actual checks removed", func() func() {
		old := use.Actual.Validation
		use.Actual.Validation = nil
		return func() { use.Actual.Validation = old }
	})
	reject("check provenance changed", func() func() {
		checks := use.Actual.WithoutRootNull().Validation
		old := a.validationSources[checks]
		a.validationSources[checks] = "/different"
		return func() { a.validationSources[checks] = old }
	})
	reject("enum vocabulary changed", func() func() {
		old := use.Enum.Members[0]
		use.Enum.Members[0].Wire = "different"
		return func() { use.Enum.Members[0] = old }
	})
	reject("actual collection element replaced", func() func() {
		rows := componentDeclaration(declarations, "RowsDto").(*ValueAlias)
		inner := rows.Type.Element
		old := inner.Element
		copy := *old
		inner.Element = &copy
		return func() { inner.Element = old }
	})
	reject("use dropped", func() func() {
		old := source.Uses
		source.Uses = nil
		return func() { source.Uses = old }
	})
	if _, failure := a.inlineEnumCollectionChildDefault(node, source.Pointer, MaxSchemaDepth+1); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("cached source bypassed depth limit", failure)
	}
}
