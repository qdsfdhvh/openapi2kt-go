package kotlin

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/internal/schema"
)

func budgetTree(minimum string) schema.TypeRef {
	leaf := schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.Int", Validation: &schema.PrimitiveValidation{Minimum: &schema.NumericBound{Literal: minimum}}}
	for range 16 {
		child := leaf
		leaf = schema.TypeRef{Kind: schema.List, Element: &child}
	}
	return leaf.Null()
}

func TestWideDeepChecksShareOnlyIdenticalTypedBodies(t *testing.T) {
	model := &schema.Model{ClassName: "Wide"}
	for i := range 200 {
		model.Properties = append(model.Properties, schema.Property{Name: fmt.Sprintf("p%d", i), Type: budgetTree("1")})
	}
	names := primitiveHelperNames(model, nil)
	unique := map[string]bool{}
	for _, name := range names {
		unique[name] = true
	}
	if len(names) != 200 || len(unique) != 1 {
		t.Fatal("wide/deep checks escaped shared typed body", len(names), len(unique))
	}
	model.Properties[199].Type = budgetTree("2")
	names = primitiveHelperNames(model, nil)
	if names[0] == names[199] {
		t.Fatal("different retained bounds shared a body")
	}
	model.Properties[198].Type = model.Properties[198].Type.WithoutRootNull()
	names = primitiveHelperNames(model, nil)
	if names[0] == names[198] {
		t.Fatal("different Kotlin nullable parameter types shared a body")
	}
	model.Properties[0].Name = "openapiKtValidatePrimitive"
	names = primitiveHelperNames(model, nil)
	if names[0] == "openapiKtValidatePrimitive" {
		t.Fatal("helper collided with original property")
	}
	var out strings.Builder
	emitModelValidation(model, 1, nil, &out)
	if !strings.Contains(out.String(), "value >= 1") || !strings.Contains(out.String(), "value >= 2") || strings.Count(out.String(), "private fun ") != 3 {
		t.Fatal("typed helper sharing dropped distinct type/check plans")
	}
}

func TestSmallDeepCheckAndScalarChecksKeepTheirExistingRoute(t *testing.T) {
	model := &schema.Model{ClassName: "Small", Properties: []schema.Property{{Name: "rows", Type: budgetTree("1")}}}
	if len(primitiveHelperNames(model, nil)) != 0 {
		t.Fatal("independent depth profile changed")
	}
	model = &schema.Model{ClassName: "Wide"}
	for i := range 120 {
		model.Properties = append(model.Properties, schema.Property{Name: fmt.Sprintf("p%d", i), Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.Long"}, Validation: &schema.PrimitiveValidation{Minimum: &schema.NumericBound{Literal: "0L"}, Maximum: &schema.NumericBound{Literal: "1L"}}})
	}
	if len(primitiveHelperNames(model, nil)) != 0 {
		t.Fatal("non-traversing scalar checks changed route")
	}
}
