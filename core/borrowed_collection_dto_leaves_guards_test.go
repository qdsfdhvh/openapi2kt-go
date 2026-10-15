package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBorrowedCollectionDTOLeavesFullSourceCacheAndNominalRecursion(t *testing.T) {
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
			doc := schemas(`{"Node":{"properties":{"id":{"type":"string"},"children":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}},"required":["id"]},"Source":{"properties":{` + strings.Join(props, ",") + `}},` + root + `"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`)
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
		doc := schemas(`{"A":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/rows"}}}},"B":{"properties":{"a":{"$ref":"#/components/schemas/A"}}},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/B"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("actual nominal mutual recursion rejected", checked, err)
		}
	}
}
func TestBorrowedCollectionDTOLeavesDiscardedConstOnlyAndOtherRoles(t *testing.T) {
	for _, node := range []string{`{}`, `{"properties":{"tag":{"type":"string","const":"fixed"}}}`, `{"type":"string","enum":["ready"]}`, `{"type":"string","x-kotlin-value-class":true}`, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`} {
		doc := schemas(`{"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Node":` + node + `}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if node == `{"oneOf":[{"type":"string"},{"type":"integer"}]}` {
			borrowedUnionGraduation(t, doc, "org.example.models.models.NodeDto")
			continue
		}
		if node == `{"type":"string","enum":["ready"]}` || node == `{"type":"string","x-kotlin-value-class":true}` {
			// Preserve the former excluded input as this independent enum profile positive.
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "kotlin.collections.List<org.example.models.models.NodeDto>") {
				t.Fatal("original string-enum collection graduation lost canonical vocabulary", err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("nonordinary borrowed leaf authorized", node, err)
		}
	}
	// A discarded borrowed shape cannot own the referenced original nested DTO.
	doc := schemas(`{"Owner":{"allOf":[{"properties":{"node":{"properties":{"id":{"type":"string"}}}}},{"properties":{"node":{"type":"string"}}}]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Owner/allOf/0/properties/node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
	got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
	if err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("discarded leaf constructed fictitious owner", err)
	}
}
