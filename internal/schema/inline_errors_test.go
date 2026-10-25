package schema

import (
	"strings"
	"testing"

	"openapi-kt/internal/document"
)

func TestInlineErrorSourceAndRoleProvenance(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","components":{"responses":{"Problem":{"content":{"application/json":{"schema":{"properties":{"message":{"type":"string"}}}}}}}},"paths":{"/a":{"get":{"operationId":"alpha","responses":{"204":{},"400":{"$ref":"#/components/responses/Problem"}}}}}}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	plan, _, failure := Analyze(root, Options{PackageName: "org.example", Suffix: "Dto"})
	after, _ := root.MarshalJSON()
	if failure != nil || string(before) != string(after) {
		t.Fatal("borrowed error source mutated", failure)
	}
	if plan.operationRoles["AlphaError400Dto"] != "Error400Dto" || len(plan.operationOrigins["AlphaError400Dto"]) != 1 || !strings.HasPrefix(plan.operationOrigins["AlphaError400Dto"][0], "/components/responses/Problem/content/") {
		t.Fatal("error provenance lost", plan.operationRoles, plan.operationOrigins)
	}
	if len(plan.Client.Models) != 1 {
		t.Fatal(plan.Client.Models)
	}
}

func TestInlineErrorRootContainerSourceProvenance(t *testing.T) {
	input := []byte(`{"openapi":"3.1.0","paths":{"/a":{"get":{"operationId":"alpha","responses":{"204":{},"400":{"content":{"application/json":{"schema":{"type":"array","items":{"allOf":[{"properties":{"message":{"type":"string"}}},{"properties":{"code":{"type":"integer"}}}]}}}}}}}}}}`)
	root, err := document.ParseObject(input, len(input))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := root.MarshalJSON()
	plan, _, failure := Analyze(root, Options{PackageName: "org.example", Suffix: "Dto"})
	after, _ := root.MarshalJSON()
	if failure != nil || string(before) != string(after) {
		t.Fatal("mutated original container source", failure)
	}
	if len(plan.Client.Models) != 1 || plan.Client.Models[0].Name() != "AlphaError400DtoInner" {
		t.Fatal("lost actual inner declaration", plan.Client.Models)
	}
	ty := plan.Client.Groups[0].Operations[0].Errors[0].Type
	if ty == nil || ty.Kind != List || ty.Element.Name != "org.example.api.models.AlphaError400DtoInner" {
		t.Fatal("manufactured root wrapper or unqualified item type", ty)
	}
}
