package schema

import (
	"slices"
	"testing"

	"openapi-kt/internal/document"
)

func closedUnionTestAnalyzer(t *testing.T, input []byte) (*analyzer, *document.Value, []Declaration, []referencedFieldDefault) {
	t.Helper()
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	pending := []referencedFieldDefault{}
	a := &analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto", Validation: true}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
	declarations, failure := a.analyze(root)
	if failure != nil {
		t.Fatal(failure)
	}
	if failure := a.validateInternalModels(declarations); failure != nil {
		t.Fatal(failure)
	}
	return a, root, declarations, pending
}

func TestClosedUnionPendingTypeAuthorityBeforeSuppression(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"oneOf":[{"properties":{"value":{"type":"string"}},"required":["value"]},{"properties":{"value":{"type":"integer"}},"required":["value"]}]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Choice","default":{"type":"Variant1","value":"x"}}}}}}}`)
	a, _, _, pending := closedUnionTestAnalyzer(t, input)
	original := pending[0]
	for _, mode := range []string{"optional", "required", "Patch", "discarded"} {
		for _, wrong := range []TypeRef{{Kind: Scalar, Name: "kotlin.String"}, {Kind: Named, Name: "other.ChoiceDto"}} {
			plan := original
			model := *original.Model
			model.Properties = slices.Clone(original.Model.Properties)
			plan.Model = &model
			if mode == "discarded" {
				plan.Model = nil
			} else if mode == "required" {
				model.Properties[plan.Index].DefaultNull = false
			} else if mode == "Patch" {
				element := original.Type.WithoutRootNull()
				model.Properties[plan.Index].Type = TypeRef{Kind: PatchValue, Name: "org.example.defaults.models.Patch", Element: &element}
				model.Properties[plan.Index].DefaultNull = false
			}
			plan.Type = wrong
			_, failure := a.planReferencedUnionDefault(plan)
			if failure == nil || failure.Code != "E_SCHEMA_UNSUPPORTED" || failure.Pointer == nil || *failure.Pointer != "/components/schemas/Holder/properties/v/default" {
				t.Fatal("wrong pending type bypassed source authority", mode, wrong, failure)
			}
		}
	}
}

func TestClosedUnionDefaultSourceAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Choice":{"oneOf":[{"properties":{"value":{"type":"string"}},"required":["value"]},{"properties":{"value":{"type":"integer"}},"required":["value"]}]},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Choice","default":{"value":"x","type":"Variant1"}}}}}}}`)
	a, root, declarations, pending := closedUnionTestAnalyzer(t, input)
	if len(pending) != 1 {
		t.Fatal("unexpected plans", len(pending))
	}
	plan := pending[0]
	source := get(referencedSchemas(root), "Choice")
	var union *ObjectUnion
	// Inspect the retained tree before default replanning registers any scratch
	// origins. The origins map alone is not a retained declaration inventory.
	for _, declaration := range declarations {
		if declaration.Name() == "ChoiceDto" {
			union, _ = declaration.(*ObjectUnion)
		}
	}
	if union == nil || union.Content() || !slices.Equal(union.SerialNames, []string{"Variant1", "Variant2"}) || a.unionOrigins[union].Schema != source {
		t.Fatal("did not retain actual closed declaration")
	}
	branches := get(source, "oneOf").Items()
	if a.modelOrigins[union.Variants[0]] != branches[0] || a.modelPointers[union.Variants[0]] != "/components/schemas/Choice/oneOf/0" {
		t.Fatal("lost original variant node/Pointer")
	}
	value := get(plan.Field.Schema, "default")
	encoded, failure := a.objectUnionEncoded(value, "/v/default", source, "/components/schemas/Choice", plan.Field.Depth, union)
	if failure != nil || encoded != `{"type":"Variant1","value":"x"}` {
		t.Fatal(encoded, failure)
	}
	a.nominalsLinked = false
	if _, failure := a.objectUnionEncoded(value, "/v/default", source, "/components/schemas/Choice", plan.Field.Depth, union); failure == nil {
		t.Fatal("provisional source authorized a closed default")
	}
	a.nominalsLinked = true
	a.hierarchySources[source] = true
	if _, failure := a.objectUnionEncoded(value, "/v/default", source, "/components/schemas/Choice", plan.Field.Depth, union); failure == nil {
		t.Fatal("hierarchy source authorized a closed default")
	}
	delete(a.hierarchySources, source)
	wrong := *union
	wrong.ClassName = "Validation"
	if _, failure := a.objectUnionEncoded(value, "/v/default", source, "/components/schemas/Choice", plan.Field.Depth, &wrong); failure == nil {
		t.Fatal("scratch owner name authorized a closed default")
	}
	original, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := original.MarshalJSON()
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("rewrote source schema or default metadata")
	}
}
