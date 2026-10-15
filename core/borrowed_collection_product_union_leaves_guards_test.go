package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionProductUnionLeavesNullViewsAndNoDrop(t *testing.T) {
	product := borrowedProductUnionSource()
	doc := schemas(`{"Node":` + product + `,"Maybe":{"$ref":"#/components/schemas/Node","nullable":true},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":["array","null"],"items":{"$ref":"#/components/schemas/Maybe"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/False"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},"Narrow":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe","nullable":false}}},"required":["rows"]}}`)
	for _, checked := range []bool{false, true} {
		rq := request(doc, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "BridgeDto"), "List<org.example.models.models.NodeDto?>") || !strings.Contains(source(t, got, "NarrowDto"), "List<org.example.models.models.NodeDto>") {
			t.Fatal("product container and item intent merged")
		}
	}
	graduatedNode := `{"allOf":[{"oneOf":[{"properties":{"id":{"type":"string","default":"bad"}}},{"properties":{"name":{"type":"string"}}}]},{"properties":{"extra":{"type":"string"}}}]}`
	graduated := request(schemas(`{"Node":`+graduatedNode+`,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`), "Dto")
	graduated.Options.Validation = true
	accepted, e := core.GenerateWithDiagnostics(graduated)
	if e != nil || !strings.Contains(source(t, accepted, "NodeDto"), `id: kotlin.String? = "bad",`) || !strings.Contains(source(t, accepted, "BridgeDto"), "NodeDto") {
		t.Fatal("original product default field lost", accepted, e)
	}
	for _, node := range []string{
		`{"allOf":[{"oneOf":[{"type":"string"},{"type":"boolean"}]},{"properties":{"id":{"type":"string"}}}]}`,
		`{"allOf":[{"oneOf":[{"properties":{"id":{"type":"string","minLength":-1}}},{"properties":{"name":{"type":"string"}}}]},{"properties":{"extra":{"type":"string"}}}]}`,
	} {
		d := schemas(`{"Node":` + node + `,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		rq := request(d, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("product original unsupported grammar discarded", node, err)
		}
	}
}

func TestBorrowedCollectionProductUnionLeavesSourceAndCheckedBudgets(t *testing.T) {
	for _, hops := range []int{43, 44} {
		props := []string{`"terminal":{"$ref":"#/components/schemas/Node"}`}
		next := "terminal"
		for i := 0; i < hops; i++ {
			key := fmt.Sprintf("h%d", i)
			props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
			next = key
		}
		props = append(props, `"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
		doc := schemas(`{"Node":` + borrowedProductUnionSource() + `,"Source":{"properties":{` + strings.Join(props, ",") + `}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/rows"}},"required":["value"]}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if hops == 43 {
			if err != nil {
				t.Fatal("preceding product source48 rejected", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("product nominal barrier erased preceding source49", err)
		}
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		doc := schemas(`{"Node":` + borrowedProductUnionSource() + `,"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"value":` + leaf + `},"required":["value"]}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("product nominal leaf erased checked17", err)
		}
	}
}

func TestBorrowedCollectionProductUnionLeavesBranchBudget(t *testing.T) {
	for _, count := range []int{240, 241} {
		branches := []string{}
		for i := 0; i < count; i++ {
			branches = append(branches, fmt.Sprintf(`{"properties":{"kind":{"const":"k%d"},"id":{"type":"string"}},"required":["id"]}`, i))
		}
		doc := schemas(`{"Node":{"allOf":[{"oneOf":[` + strings.Join(branches, ",") + `]},{"properties":{"stamp":{"const":"shared","type":"string"}}}]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("product241 budget lost at borrowed leaf", err)
		}
	}
}
