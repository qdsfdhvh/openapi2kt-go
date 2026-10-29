package schema

import "testing"

func TestReferencedEnumCollectionChildAuthority(t *testing.T) {
	for _, checked := range []bool{false, true} {
		root := loadComponentDocument(t, `{"Word":{"enum":["one","two"]},"Other":{"enum":["one","two"]},"WordAlias":{"$ref":"#/components/schemas/Word"},"Values":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias"}},"ValuesAlias":{"$ref":"#/components/schemas/Values"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias","default":["one"]}}}`)
		pending := []referencedFieldDefault{}
		a := &analyzer{
			root: root, referencedDefaults: &pending,
			options:    Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked},
			components: map[string]*component{}, emissionHints: EmissionHints{},
			validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
			aliasUses: map[collectionAliasUse]struct{}{},
		}
		declarations, err := a.analyze(root)
		if err != nil {
			t.Fatal("analyze", checked, err)
		}
		if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
			t.Fatal("freeze", checked, err)
		}
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("original authority", checked, err)
		}
		source := a.collectionChildDefaults[get(a.components["Rows"].Schema, "items")]
		if source == nil || !source.ReferencedEnumCollection || len(source.Collections) != 2 || len(source.ScalarSources) != 2 {
			t.Fatal("missing enum collection and alias proofs", checked, source)
		}
		peer := componentDeclaration(declarations, a.components["Word"].Name).(*StringEnum)
		reject := func(name string, mutate func() func()) {
			t.Helper()
			restore := mutate()
			if failure := a.validateInternalModels(declarations); failure == nil {
				t.Error("changed enum reference retained authority", checked, name)
			}
			restore()
			if failure := a.validateInternalModels(declarations); failure != nil {
				t.Fatal("restored authority rejected", checked, name, failure)
			}
		}
		reject("enum vocabulary", func() func() {
			old := peer.Members[0]
			peer.Members[0].Wire = "future"
			return func() { peer.Members[0] = old }
		})
		reject("enum origin", func() func() {
			old := a.enumOrigins[peer]
			changed := old
			changed.Pointer += "/copy"
			a.enumOrigins[peer] = changed
			return func() { a.enumOrigins[peer] = old }
		})
		reject("same-type enum alias rebinding", func() func() {
			alias := a.components["WordAlias"]
			old := alias.BindingTarget
			alias.BindingTarget = a.components["Other"]
			return func() { alias.BindingTarget = old }
		})
		reject("copied enum export", func() func() {
			for i, d := range declarations {
				if d == peer {
					copy := *peer
					declarations[i] = &copy
					return func() { declarations[i] = peer }
				}
			}
			t.Fatal("missing enum export")
			return nil
		})
		reject("duplicated enum export", func() func() {
			declarations = append(declarations, peer)
			return func() { declarations = declarations[:len(declarations)-1] }
		})
	}
}

func TestReferencedEnumCollectionChildIntermediateAuthority(t *testing.T) {
	for _, checked := range []bool{false, true} {
		root := loadComponentDocument(t, `{"Word":{"enum":["one","two"]},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},"Other":{"type":"array","items":{"$ref":"#/components/schemas/Word"}},"Middle":{"$ref":"#/components/schemas/Inner"},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Middle"}},"ValuesAlias":{"$ref":"#/components/schemas/Values"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias","default":[["one"]]}}}`)
		pending := []referencedFieldDefault{}
		a := &analyzer{
			root: root, referencedDefaults: &pending,
			options:    Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked},
			components: map[string]*component{}, emissionHints: EmissionHints{},
			validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
			aliasUses: map[collectionAliasUse]struct{}{},
		}
		declarations, err := a.analyze(root)
		if err != nil {
			t.Fatal("analyze", checked, err)
		}
		if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
			t.Fatal("freeze", checked, err)
		}
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("original authority", checked, err)
		}
		source := a.collectionChildDefaults[get(a.components["Rows"].Schema, "items")]
		if source == nil || !source.ReferencedEnumCollection || len(source.Collections) != 4 {
			t.Fatal("missing intermediate collection proofs", checked, source)
		}
		reject := func(name string, mutate func() func()) {
			t.Helper()
			restore := mutate()
			if failure := a.validateInternalModels(declarations); failure == nil {
				t.Error("changed intermediate retained authority", checked, name)
			}
			restore()
			if failure := a.validateInternalModels(declarations); failure != nil {
				t.Fatal("restored intermediate rejected", checked, name, failure)
			}
		}
		middle := a.components["Middle"]
		reject("same-type rebinding", func() func() {
			old := middle.BindingTarget
			middle.BindingTarget = a.components["Other"]
			return func() { middle.BindingTarget = old }
		})
		reject("cached collection height", func() func() {
			old := middle.CollectionHeight
			middle.CollectionHeight++
			return func() { middle.CollectionHeight = old }
		})
		reject("nullable binding", func() func() {
			old := middle.BindingNullable
			value := true
			middle.BindingNullable = &value
			return func() { middle.BindingNullable = old }
		})
		reject("copied intermediate declaration", func() func() {
			for i, declaration := range declarations {
				if declaration.Name() == middle.Name {
					old := declarations[i]
					copy := *old.(*ValueAlias)
					declarations[i] = &copy
					return func() { declarations[i] = old }
				}
			}
			t.Fatal("missing intermediate declaration")
			return nil
		})
		reject("copied intermediate backing", func() func() {
			alias := componentDeclaration(declarations, a.components["Inner"].Name).(*ValueAlias)
			old := alias.Type.Element
			copy := *old
			alias.Type.Element = &copy
			return func() { alias.Type.Element = old }
		})
	}
}

func TestReferencedNativeEnumCollectionChildAuthority(t *testing.T) {
	for _, checked := range []bool{false, true} {
		root := loadComponentDocument(t, `{"Code":{"type":"integer","enum":[1,2]},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Code"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[1]}}}`)
		pending := []referencedFieldDefault{}
		a := &analyzer{
			root: root, referencedDefaults: &pending,
			options:    Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: checked},
			components: map[string]*component{}, emissionHints: EmissionHints{},
			validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
			aliasUses: map[collectionAliasUse]struct{}{},
		}
		declarations, err := a.analyze(root)
		if err != nil {
			t.Fatal("analyze", err)
		}
		if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
			t.Fatal("freeze", err)
		}
		if err := a.validateInternalModels(declarations); err != nil {
			t.Fatal("original authority", err)
		}
		peer := componentDeclaration(declarations, a.components["Code"].Name).(*ScalarAlias)
		reject := func(name string, mutate func() func()) {
			t.Helper()
			restore := mutate()
			if failure := a.validateInternalModels(declarations); failure == nil {
				t.Error("changed native enum retained authority", name)
			}
			restore()
			if failure := a.validateInternalModels(declarations); failure != nil {
				t.Fatal("restored authority rejected", name, failure)
			}
		}
		reject("native codec", func() func() {
			old := peer.Scalar
			peer.Scalar = "kotlin.String"
			return func() { peer.Scalar = old }
		})
		reject("native factory nullability", func() func() {
			old := peer.NullableComponentDefault
			peer.NullableComponentDefault = !old
			return func() { peer.NullableComponentDefault = old }
		})
		reject("copied native export", func() func() {
			for i, d := range declarations {
				if d == peer {
					copy := *peer
					declarations[i] = &copy
					return func() { declarations[i] = peer }
				}
			}
			t.Fatal("missing native export")
			return nil
		})
		reject("duplicated native export", func() func() {
			declarations = append(declarations, peer)
			return func() { declarations = declarations[:len(declarations)-1] }
		})
		source := a.collectionChildDefaults[get(a.components["Rows"].Schema, "items")]
		reject("native origin proof", func() func() {
			old := source.ReferencedEnumNative
			source.ReferencedEnumNative = nil
			return func() { source.ReferencedEnumNative = old }
		})
	}
}

func TestReferencedNullableNativeEnumCollectionChildAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"Code":{"type":"integer","nullable":true,"enum":[1,2],"default":null},"Values":{"type":"array","items":{"$ref":"#/components/schemas/Code"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":[null,1]}}}`)
	pending := []referencedFieldDefault{}
	a := &analyzer{
		root: root, referencedDefaults: &pending,
		options:    Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true},
		components: map[string]*component{}, emissionHints: EmissionHints{},
		validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{},
		aliasUses: map[collectionAliasUse]struct{}{},
	}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal("analyze", err)
	}
	if err := a.finalizeCollectionReferenceChildDefaults(declarations, nil); err != nil {
		t.Fatal("freeze", err)
	}
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("original authority", err)
	}
	peer := componentDeclaration(declarations, a.components["Code"].Name).(*ValueAlias)
	if peer.Type.Element == nil {
		t.Fatal("missing nullable native backing")
	}
	old := peer.Type.Element
	copy := *old
	peer.Type.Element = &copy
	if a.validateInternalModels(declarations) == nil {
		t.Error("copied nullable native backing retained authority")
	}
	peer.Type.Element = old
	if err := a.validateInternalModels(declarations); err != nil {
		t.Fatal("restored nullable native backing rejected", err)
	}
}
