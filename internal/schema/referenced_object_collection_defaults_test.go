package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func referencedCollectionTestAnalyzer(t *testing.T, input []byte) (*analyzer, *document.Value) {
	t.Helper()
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	pending := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	return a, root
}

func TestReferencedObjectCollectionDefaultInlineElementAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7}}},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Item","nullable":true},"default":[{"name":"a"},null]}}}}}}`)
	a, root := referencedCollectionTestAnalyzer(t, input)
	field := get(get(get(referencedSchemas(root), "Holder"), "properties"), "value")
	if !a.objectDefaultCandidate(field) {
		t.Fatal("inline referenced object element not recognized")
	}
	node, failure := a.objectDefaultNode(field, "/components/schemas/Holder/properties/value", 1, true)
	if failure != nil {
		t.Fatal(failure)
	}
	item := get(referencedSchemas(root), "Item")
	if node.ty.Kind != List || node.element == nil || !node.element.object || node.element.schema != item {
		t.Fatal("element did not borrow the referenced DTO target", node.ty, node.element)
	}
	if node.element.ty.Kind != Nullable {
		t.Fatal("explicit element nullability override lost", node.element.ty)
	}
	canonical, failure := node.value(get(field, "default"), "/components/schemas/Holder/properties/value/default")
	if failure != nil || canonical != `[{"name":"a"},null]` {
		t.Fatal(canonical, failure)
	}
	// A `$ref` to a scalar is not an object collection candidate.
	scalarField := get(get(get(referencedSchemas(root), "Holder"), "properties"), "value")
	_ = scalarField
	scalar := []byte(`{"openapi":"3.1.0","components":{"schemas":{"N":{"type":"integer"},"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/N"}}}}}}}`)
	b, sroot := referencedCollectionTestAnalyzer(t, scalar)
	if b.objectDefaultCandidate(get(get(get(referencedSchemas(sroot), "Holder"), "properties"), "value")) {
		t.Fatal("scalar reference recognized as object element")
	}
}

func TestReferencedObjectCollectionDefaultNamedTargetAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Item":{"type":"object","properties":{"name":{"type":"string"}}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows","default":[{"name":"a"}]}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := func() (*analyzer, *document.Value) {
		root, err := document.Load(input, document.JSON, "")
		if err != nil {
			t.Fatal(err)
		}
		an := &analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
		ds, err := an.analyze(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := an.validateInternalModels(ds); err != nil {
			t.Fatal(err)
		}
		if err := an.planReferencedFieldDefaults(); err != nil {
			t.Fatal(err)
		}
		return an, root
	}()
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	plan := pending[0]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if err != nil {
		t.Fatal(err)
	}
	item := get(referencedSchemas(root), "Item")
	if node.ty.Kind != List || node.element == nil || !node.element.object || node.element.schema != item {
		t.Fatal("named collection element did not borrow the DTO target", node.ty, node.element)
	}
	if got := plan.Model.Properties[plan.Index].Default; !strings.Contains(got, "`openapi-kt default tree Alias") {
		t.Fatal("named collection helper not produced", got)
	}
}
