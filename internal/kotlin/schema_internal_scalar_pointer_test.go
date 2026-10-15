package kotlin

import (
	"strings"
	"testing"

	"openapi-kt/internal/schema"
)

func TestSchemaInternalScalarPointerTypedHelperDemandAndNames(t *testing.T) {
	n := int64(1)
	checks := &schema.PrimitiveValidation{MinLength: &n}
	leaf := schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String", Validation: checks}
	ty := leaf.Null()
	model := &schema.Model{ClassName: "OpenapiKtValidatePrimitive", Properties: []schema.Property{
		{Name: "v", WireName: "v", Type: ty},
		{Name: "openapiKtValidatePrimitive2", WireName: "openapiKtValidatePrimitive2", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String"}},
	}, Nested: []schema.Declaration{&schema.Model{ClassName: "OpenapiKtValidatePrimitive3"}}, Constants: []schema.Constant{{Name: "openapiKtValidatePrimitive4", WireName: "openapiKtValidatePrimitive4", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String"}, Initializer: `"c"`}}, PropertyCount: &schema.CollectionValidation{Minimum: &n, MinimumKey: "minProperties"}}
	emit := func(hints schema.EmissionHints) string {
		var b strings.Builder
		emitModelValidation(model, 1, hints, &b)
		return b.String()
	}
	plain := emit(nil)
	if strings.Contains(plain, "private fun") || !strings.Contains(plain, "this.v?.let") {
		t.Fatal("unhinted typed output changed", plain)
	}
	hinted := emit(schema.EmissionHints{checks: true})
	for _, want := range []string{"this.openapiKtValidatePrimitive5(this.v)", "private fun openapiKtValidatePrimitive5(value: kotlin.String?)", "value?.let", "Field value is shorter than minLength."} {
		if !strings.Contains(hinted, want) {
			t.Fatal(want, hinted)
		}
	}
	for _, bad := range []string{"private inline", "private val", "Transient", "serializer"} {
		if strings.Contains(hinted, bad) {
			t.Fatal("extra state or altered serializer", hinted)
		}
	}
	// Full typed property, not an extracted scalar, is the helper input. Keep
	// collection/count/unique/null wrappers in the unchanged recursive checker.
	item := leaf.Null()
	list := schema.TypeRef{Kind: schema.List, Element: &item, Collection: &schema.CollectionValidation{Minimum: &n, MinimumKey: "minItems", Unique: true}}
	model.Properties[0].Type = schema.TypeRef{Kind: schema.PatchValue, Name: "org.example.models.Patch", Element: &list}
	text := emit(schema.EmissionHints{checks: true})
	for _, want := range []string{"value: org.example.models.Patch<kotlin.collections.List<kotlin.String?>>", "value.valueOrNull?.let", "minItems", "`openapi-kt uniqueItems`", "forEach", "value?.let", "minLength"} {
		if !strings.Contains(text, want) {
			t.Fatal(want, text)
		}
	}
	if model.Properties[0].Validation != nil || model.Properties[0].Type.Element.Element.Element.Validation != checks {
		t.Fatal("typed validation representation moved")
	}
}
