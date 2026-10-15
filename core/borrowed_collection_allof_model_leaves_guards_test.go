package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBorrowedCollectionAllOfModelLeavesExcludedPhysicalOwnersAndNullable(t *testing.T) {
	for _, node := range []string{`{"allOf":[{"type":"object"}]}`, `{"allOf":[{"properties":{"tag":{"type":"string","const":"fixed"}}}]}`, `{"allOf":[{"oneOf":[{"properties":{"tag":{"const":"a"},"id":{"type":"string"}}},{"properties":{"tag":{"const":"b"},"name":{"type":"string"}}}]},{"properties":{"extra":{"type":"string"}}}]}`} {
		doc := schemas(`{"Leaf":` + node + `,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		if strings.Contains(node, `"oneOf"`) {
			borrowedProductGraduation(t, doc, "org.example.models.models.LeafDto")
			continue
		}
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if node == `{"allOf":[{"type":"object"}]}` {
			// Keep the historical empty allOf input as an explicit raw-leaf positive.
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "List<kotlinx.serialization.json.JsonElement>") {
				t.Fatal("original empty Model lost transparent raw backing", err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("nonconstructor allOf physical source authorized", node, err)
		}
	}
	for _, checked := range []bool{false, true} {
		doc := schemas(`{"Owner":{"properties":{"leaf":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"],"nullable":true},{"properties":{"label":{"type":"string"}}}]}}},"LeafAlias":{"$ref":"#/components/schemas/Owner/properties/leaf"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/LeafAlias"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Narrow":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/LeafAlias","nullable":false}}}}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "BridgeDto"), "List<org.example.models.models.OwnerDto.Leaf?>") || !strings.Contains(source(t, got, "NarrowDto"), "List<org.example.models.models.OwnerDto.Leaf>") {
			t.Fatal("merged nullable leaf source or explicit false lost")
		}
	}
}

func TestBorrowedCollectionAllOfModelLeavesForwardShapeAliasesAndCheckedDepth(t *testing.T) {
	forward := schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Node":{"allOf":[{"$ref":"#/components/schemas/BaseAgain"},{"properties":{"label":{"type":"string"}}}]},"BaseAgain":{"$ref":"#/components/schemas/BaseAlias"},"BaseAlias":{"$ref":"#/components/schemas/Base"},"Base":{"properties":{"id":{"type":"string"}},"required":["id"]}}`)
	if _, err := core.GenerateWithDiagnostics(request(forward, "Dto")); err != nil {
		t.Fatal("forward original shape alias graph rejected", err)
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		doc := schemas(`{"Node":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"label":{"type":"string"}}}]},"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"rows":` + leaf + `},"required":["rows"]}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("allOf leaf collection checked17 authorized", err)
		}
	}
}

func TestBorrowedCollectionAllOfModelLeavesFullSourceCacheAndNominalRecursion(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, hops := range []int{43, 44} {
			props := []string{`"terminal":{"$ref":"#/components/schemas/Node"}`}
			next := "terminal"
			for i := 0; i < hops; i++ {
				key := fmt.Sprintf("h%d", i)
				props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
				next = key
			}
			props = append(props, `"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
			field := `{"$ref":"#/components/schemas/Source/properties/rows"}`
			root := ""
			if cached {
				root = `"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},`
				field = `{"$ref":"#/components/schemas/Bridge","nullable":false}`
			}
			doc := schemas(`{"Node":{"allOf":[{"properties":{"id":{"type":"string"}},"required":["id"]},{"properties":{"children":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}}]},"Source":{"properties":{` + strings.Join(props, ",") + `}},` + root + `"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`)
			r := request(doc, "Dto")
			got, err := core.GenerateWithDiagnostics(r)
			if hops == 43 && !cached {
				if err != nil {
					t.Fatal("full original source48 rejected", cached, hops, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("nominal erased preceding collectionwork", cached, hops, err)
			}
		}
	}
	for _, checked := range []bool{false, true} {
		doc := schemas(`{"A":{"allOf":[{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/rows"}}}},{"properties":{"id":{"type":"string"}}}]},"B":{"allOf":[{"properties":{"a":{"$ref":"#/components/schemas/A"}}},{"properties":{"id":{"type":"string"}}}]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/B"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("actual nominal mutual recursion rejected", checked, err)
		}
	}
}
