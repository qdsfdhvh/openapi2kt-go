package schema

import (
	"slices"
	"testing"
)

func TestComposedEnumOwnDefaultsActualConstructorSourceAuthority(t *testing.T) {
	root := loadComponentDocument(t, `{"U":{"allOf":[{"required":["version"],"properties":{"version":{"const":"v1"}}},{"oneOf":[{"required":["kind"],"properties":{"kind":{"const":"a"},"mode":{"type":"string","enum":["on","off"],"default":"on"},"number":{"type":"integer","format":"int32","enum":[1,2],"default":1}}},{"required":["kind"],"properties":{"kind":{"const":"b"},"other":{"type":"string"}}}]}]}}`)
	before, _ := root.MarshalJSON()
	a := analyzer{root: root, options: Options{PackageName: "org.example.composed", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}}
	declarations, err := a.analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	union, ok := componentDeclaration(declarations, "UDto").(*ObjectUnion)
	if !ok || len(union.Variants) != 2 {
		t.Fatal("missing actual Cartesian variant declarations", declarations)
	}
	component := a.components["U"]
	composition, err := a.objectCompositionUnion(component.Schema, component.Pointer, 0)
	if err != nil {
		t.Fatal(err)
	}
	shape, variant := composition.Branches[0].Shape, union.Variants[0]
	work := 0
	a.objectDefaultWork = &work
	node, err := a.composedVariantDefaultNode(shape, variant)
	if err != nil {
		t.Fatal(err)
	}
	steps := work
	if steps < len(shape.Fields)+1 {
		t.Fatal("selected variant source work was not charged", steps)
	}
	for _, child := range node.objectFields {
		field := shapeField(shape, child.wire)
		if field == nil || child.node.schema != field.Schema || child.node.pointer != field.Pointer {
			t.Fatal("selected member replaced original source node or Pointer", child.wire)
		}
	}
	for _, spare := range []int{steps, steps - 1} {
		work = maxNestedDefaultNodes - spare
		_, err := a.composedVariantDefaultNode(shape, variant)
		if spare == steps && (err != nil || work != maxNestedDefaultNodes) || spare < steps && (err == nil || err.Code != "E_MODEL_LIMIT") {
			t.Fatal("selected member reset shared work budget", work, err)
		}
	}
	a.objectDefaultWork = nil
	mode := slices.IndexFunc(variant.Properties, func(p Property) bool { return p.WireName == "mode" })
	if mode < 0 {
		t.Fatal("mode must belong to the actual selected constructor")
	}
	saved := variant.Properties[mode]
	variant.Properties[mode].Type = TypeRef{Kind: JSON}
	if _, err := a.composedVariantDefaultNode(shape, variant); err == nil || err.Pointer == nil || *err.Pointer != shapeField(shape, "mode").Pointer+"/default" {
		t.Fatal("raw projection incorrectly authorized an enum-own constructor role", err)
	}
	variant.Properties[mode] = saved
	origin := a.modelOrigins[variant]
	delete(a.modelOrigins, variant)
	if _, err := a.composedVariantDefaultNode(shape, variant); err == nil {
		t.Fatal("unknown constructor authorized source defaults")
	}
	a.modelOrigins[variant] = origin
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("selected variant default validation changed the original AST")
	}
}
