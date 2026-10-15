package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func referencedObjectTestAnalyzer(t *testing.T, input []byte, pending *[]referencedFieldDefault) (*analyzer, *document.Value) {
	t.Helper()
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: pending}
	ds, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	if err := a.planReferencedFieldDefaults(); err != nil {
		t.Fatal(err)
	}
	return &a, root
}

func referencedSchemas(root *document.Value) *document.Value {
	return get(get(root, "components"), "schemas")
}

func TestReferencedObjectDefaultSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Inner":{"type":"object","required":["name"],"properties":{"name":{"type":"string"},"age":{"type":"integer","default":7},"child":{"type":"object","properties":{"flag":{"type":"boolean"}},"default":{"flag":true}}}},"Alias":{"$ref":"#/components/schemas/Inner"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias","default":{"age":9,"name":"outer","child":{"flag":false}}}}}}}}`)
	pending := []referencedFieldDefault{}
	a, root := referencedObjectTestAnalyzer(t, input, &pending)
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	plan := pending[0]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if err != nil {
		t.Fatal(err)
	}
	inner := get(referencedSchemas(root), "Inner")
	if !node.object || node.schema != inner {
		t.Fatal("object node did not borrow the original target component")
	}
	if node.ty.Kind != Named || node.ty.Name != "org.example.defaults.models.InnerDto" {
		t.Fatal("effective nominal type not retained", node.ty)
	}
	holderField := get(get(get(referencedSchemas(root), "Holder"), "properties"), "value")
	canonical, failure := node.value(get(holderField, "default"), "/components/schemas/Holder/properties/value/default")
	if failure != nil || canonical != `{"age":9,"child":{"flag":false},"name":"outer"}` {
		t.Fatal(canonical, failure)
	}
	property := plan.Model.Properties[plan.Index]
	if !strings.HasPrefix(property.Default, "`openapi-kt default model`(") || !strings.Contains(property.Default, `{\"age\":9`) {
		t.Fatal("missing model default marker", property.Default)
	}
	// An invalid value keeps the consuming field's original member Pointer.
	bad, err := document.Load([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Inner":{"type":"object","properties":{"name":{"type":"string"}}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Inner","default":{"name":"x","extra":1}}}}}}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	badPending := []referencedFieldDefault{}
	badAnalyzer := analyzer{root: bad, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &badPending}
	ds, err := badAnalyzer.analyze(bad)
	if err != nil {
		t.Fatal(err)
	}
	if err := badAnalyzer.validateInternalModels(ds); err != nil {
		t.Fatal(err)
	}
	badPlan := badPending[0]
	if _, err := badAnalyzer.referencedDefaultSchema(badPlan.Field.Schema, badPlan.Field.Pointer, badPlan.Type, badPlan.Field.Depth, true); err != nil {
		t.Fatal(err)
	}
	if err := badAnalyzer.planReferencedFieldDefaults(); err == nil || err.Pointer == nil || *err.Pointer != "/components/schemas/Holder/properties/value/default/extra" {
		t.Fatal("unknown key Pointer", err)
	}
}

func TestReferencedObjectDefaultRejectsTargetOwnDefault(t *testing.T) {
	target, err := document.Load([]byte(`{"type":"object","properties":{"a":{"type":"string"}},"default":{"a":"target"}}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	a := &analyzer{options: Options{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}}
	if _, err := a.objectDefaultNode(target, "/components/schemas/Inner", 1, false); err == nil {
		t.Fatal("target own default admitted")
	}
	if _, err := a.objectDefaultNode(target, "/components/schemas/Inner", 1, true); err != nil {
		t.Fatal("property-role default key must be allowed", err)
	}
}
