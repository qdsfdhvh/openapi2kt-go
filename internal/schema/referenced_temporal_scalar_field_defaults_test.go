package schema

import (
	"testing"

	"openapi-kt/internal/document"
)

func TestReferencedTemporalScalarFieldDefaultsOriginalAuthority(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"schemas":{"AHolder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/narrow","default":"2024-02-29T12:34:56.123456789+02:30"}}},"Hops":{"properties":{"entry":{"$ref":"#/components/schemas/Alias","nullable":true},"narrow":{"$ref":"#/components/schemas/Hops/properties/entry","nullable":false}}},"Alias":{"$ref":"#/components/schemas/Stamp"},"Stamp":{"type":"string","format":"date-time","nullable":true}}}}`)
	root, err := document.Load(input, document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	pending := []referencedFieldDefault{}
	a := analyzer{root: root, options: Options{PackageName: "org.example.defaults", Suffix: "Dto"}, components: map[string]*component{}, emissionHints: EmissionHints{}, validationSources: map[*PrimitiveValidation]string{}, collectionSources: map[*CollectionValidation]string{}, aliasUses: map[collectionAliasUse]struct{}{}, referencedDefaults: &pending}
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
	if len(pending) != 1 {
		t.Fatal(len(pending))
	}
	plan := pending[0]
	canonical := a.components["Stamp"]
	node, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, plan.Type, plan.Field.Depth, true)
	if err != nil || node.schema != canonical.Schema || node.pointer != canonical.Pointer || node.ty.Kind != Scalar || node.ty.Name != "kotlin.time.Instant" || node.ty.Serializer != "" || node.ty.Validation != nil {
		t.Fatal("source/actual scalar/nearest false drift", node, err)
	}
	if plan.Model.Properties[plan.Index].Default != "`openapi-kt default instant`(\"2024-02-29T12:34:56.123456789+02:30\")" {
		t.Fatal("original lexeme lost")
	}
	target, err := a.resolveSchema("#/components/schemas/Hops/properties/narrow", "/use")
	if err != nil || target.CanonicalTemporal != canonical || target.Steps != 4 || target.Schema != canonical.Schema || target.Pointer != canonical.Pointer {
		t.Fatal("canonical/cache drift", target, err)
	}
	missing := []Declaration{}
	var original Declaration
	for _, d := range ds {
		if d.Name() == canonical.Name {
			original = d
		} else {
			missing = append(missing, d)
		}
	}
	if err := a.validateTemporalBindings(missing); err == nil {
		t.Fatal("missing declaration authorized")
	}
	copy := &ValueAlias{ClassName: canonical.Name, Type: TypeRef{Kind: Scalar, Name: canonical.Scalar}.Null()}
	if err := a.validateTemporalBindings(append(append([]Declaration{}, missing...), copy)); err == nil {
		t.Fatal("same-spelling replacement authorized")
	}
	a.temporalOrigins[copy] = a.temporalOrigins[original]
	if err := a.validateTemporalBindings(append(append([]Declaration{}, ds...), copy)); err == nil {
		t.Fatal("duplicate identity authorized")
	}
	delete(a.temporalOrigins, copy)
	saved := canonical.Schema
	cloned, err := document.Load([]byte(`{"type":"string","format":"date-time","nullable":true}`), document.JSON, "")
	if err != nil {
		t.Fatal(err)
	}
	canonical.Schema = cloned
	if err := a.validateTemporalBindings(ds); err == nil {
		t.Fatal("copied physical node authorized")
	}
	canonical.Schema = saved
	if err := a.validateTemporalBindings(ds); err != nil {
		t.Fatal(err)
	}
	// Actual serializer/type authority remains separate from source spelling.
	ty := plan.Type
	ty.Serializer = "provided.Custom"
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, ty, plan.Field.Depth, true); err == nil {
		t.Fatal("custom serializer authorized")
	}
	ty = plan.Type
	ty.Name = "kotlinx.datetime.LocalDate"
	if _, err := a.referencedDefaultSchema(plan.Field.Schema, plan.Field.Pointer, ty, plan.Field.Depth, true); err == nil {
		t.Fatal("type/source mismatch authorized")
	}
	after, _ := root.MarshalJSON()
	if string(before) != string(after) {
		t.Fatal("immutable tree changed")
	}
}
