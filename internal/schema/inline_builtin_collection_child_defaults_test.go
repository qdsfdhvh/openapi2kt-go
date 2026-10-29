package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestInlineBuiltinCollectionChildDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"integer","minimum":1,"default":2}}}}}`)
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
	if failure = a.finalizeInlineBuiltinChildDefaults(declarations, nil); failure != nil {
		t.Fatal("finalize", failure)
	}
	if failure = a.validateInlineBuiltinChildDefaults(declarations, nil, false); failure != nil {
		t.Fatal("original authority", failure)
	}
	node := get(get(get(a.components["Holder"].Schema, "properties"), "value"), "items")
	source := a.inlineBuiltinChildDefaults[node]
	if source == nil {
		t.Fatal("missing original source")
	}
	var use *inlineBuiltinChildUse
	for _, candidate := range source.Uses {
		if candidate.Retained {
			use = candidate
			break
		}
	}
	if use == nil || use.Actual.WithoutRootNull().Validation == nil {
		t.Fatal("missing retained checked element")
	}
	reject := func(name string, mutate func() func()) {
		t.Helper()
		restore := mutate()
		if failure := a.validateInlineBuiltinChildDefaults(declarations, nil, false); failure == nil {
			t.Fatal("changed source or type retained authority", name)
		}
		restore()
		if failure := a.validateInlineBuiltinChildDefaults(declarations, nil, false); failure != nil {
			t.Fatal("restored authority rejected", name, failure)
		}
	}
	reject("default node replaced", func() func() {
		for i := range node.Members() {
			if node.Members()[i].Name == "default" {
				old := node.Members()[i].Value
				copy, err := document.Load([]byte(`{"v":2}`), document.JSON, "")
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
	reject("actual element replaced", func() func() {
		holder := componentDeclaration(declarations, "HolderDto").(*Model)
		outer := &holder.Properties[0].Type
		if outer.Kind == Nullable {
			outer = outer.Element
		}
		old := outer.Element
		copy := *old
		outer.Element = &copy
		return func() { outer.Element = old }
	})
	reject("source scalar changed", func() func() {
		old := source.Scalar
		source.Scalar = "kotlin.String"
		return func() { source.Scalar = old }
	})
	reject("hierarchy owner", func() func() {
		a.hierarchySources[source.Owner] = true
		return func() { delete(a.hierarchySources, source.Owner) }
	})
	if _, failure := a.inlineBuiltinCollectionChildDefault(node, source.Pointer, MaxSchemaDepth+1); failure == nil || failure.Code != "E_SCHEMA_DEPTH" {
		t.Fatal("cached source bypassed depth limit", failure)
	}
}

func TestInlineBuiltinCollectionChildOwnerDefaultAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Holder":{"properties":{"value":{"type":"array","items":{"type":"integer","default":7}}},"default":{"value":[2]}}}`)
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
	if failure = a.finalizeInlineBuiltinChildDefaults(declarations, nil); failure != nil {
		t.Fatal("finalize", failure)
	}
	node := get(get(get(a.components["Holder"].Schema, "properties"), "value"), "items")
	source := a.inlineBuiltinChildDefaults[node]
	if source == nil || !a.componentDefaultSchemas[source.Owner] {
		t.Fatal("missing owner factory authority")
	}
	delete(a.componentDefaultSchemas, source.Owner)
	if failure = a.validateInlineBuiltinChildDefaults(declarations, nil, false); failure == nil {
		t.Fatal("unmarked owner retained child default authority")
	}
	a.componentDefaultSchemas[source.Owner] = true
	if failure = a.validateInlineBuiltinChildDefaults(declarations, nil, false); failure != nil {
		t.Fatal("restored owner factory authority rejected", failure)
	}
}
