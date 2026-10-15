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
