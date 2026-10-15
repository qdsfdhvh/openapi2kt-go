package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBorrowedCollectionUnionLeavesPhysicalCarriersAndNullable(t *testing.T) {
	for _, checked := range []bool{false, true} {
		doc := schemas(`{"Root":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]},"RootAlias":{"$ref":"#/components/schemas/Root","nullable":true},"Owner":{"properties":{"leaf":{"oneOf":[{"type":"string"},{"type":"boolean"},{"type":"null"}]}}},"LeafAlias":{"$ref":"#/components/schemas/Owner/properties/leaf"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/LeafAlias"}},"rootRows":{"type":"array","items":{"$ref":"#/components/schemas/Root"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"RootBridge":{"$ref":"#/components/schemas/Source/properties/rootRows"},"Narrow":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/LeafAlias","nullable":false}},"nullableRoot":{"type":"array","items":{"$ref":"#/components/schemas/RootAlias"}}}}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "BridgeDto"), "List<org.example.models.models.OwnerDto.Leaf?>") || !strings.Contains(source(t, got, "NarrowDto"), "List<org.example.models.models.OwnerDto.Leaf>") || !strings.Contains(source(t, got, "RootBridgeDto"), "List<org.example.models.models.RootDto>") || !strings.Contains(source(t, got, "NarrowDto"), "List<org.example.models.models.RootDto?>") {
			t.Fatal("canonical Null carrier/inline nullable leaf/nearest false lost")
		}
	}
	for _, leaf := range []string{`{"oneOf":[{"properties":{"id":{"type":"string"}}},{"type":"null"}]}`, `{"oneOf":[{"type":"object"},{"type":"null"}]}`, `{"allOf":[{"oneOf":[{"properties":{"kind":{"const":"a"},"id":{"type":"string"}}},{"properties":{"kind":{"const":"b"},"name":{"type":"string"}}}]},{"properties":{"extra":{"type":"string"}}}]}`} {
		doc := schemas(`{"Owner":{"properties":{"leaf":` + leaf + `}},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/leaf"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		if strings.Contains(leaf, `"allOf"`) {
			borrowedProductGraduation(t, doc, "org.example.models.models.OwnerDto.Leaf")
			continue
		}
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("nonretained/folded union leaf authorized", leaf, err)
		}
	}
}

func TestBorrowedCollectionUnionLeavesBranchBudgetAndMalformedSources(t *testing.T) {
	for _, count := range []int{240, 241} {
		branches := []string{}
		for i := 0; i < count; i++ {
			branches = append(branches, fmt.Sprintf(`{"properties":{"kind":{"const":"k%d"},"value":{"type":"string"}},"required":["value"]}`, i))
		}
		doc := schemas(`{"Leaf":{"oneOf":[` + strings.Join(branches, ",") + `]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_UNION_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed union erased original branch241 budget", err)
		}
	}
	for _, leaf := range []string{`{"oneOf":[{"type":"string","default":"bad"},{"type":"boolean"}]}`, `{"oneOf":[]}`, `{"oneOf":[{"type":"string"},{"type":"boolean"}],"x-native-type":"org.example.Native"}`, `{"oneOf":[{"type":"string"},{"type":"boolean"}],"discriminator":{"propertyName":"kind"}}`} {
		doc := schemas(`{"Leaf":` + leaf + `,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Leaf"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("original union grammar skipped", leaf, err)
		}
	}
}

func TestBorrowedCollectionUnionLeavesForwardShapeAliasesAndCheckedDepth(t *testing.T) {
	forward := schemas(`{"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Node":{"oneOf":[{"$ref":"#/components/schemas/BaseAgain"},{"properties":{"kind":{"const":"label"},"label":{"type":"string"}},"required":["label"]}]},"BaseAgain":{"$ref":"#/components/schemas/BaseAlias"},"BaseAlias":{"$ref":"#/components/schemas/Base"},"Base":{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]}}`)
	if _, err := core.GenerateWithDiagnostics(request(forward, "Dto")); err != nil {
		t.Fatal("forward original shape alias graph rejected", err)
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		doc := schemas(`{"Node":{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]},{"properties":{"kind":{"const":"label"},"label":{"type":"string"}},"required":["label"]}]},"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"rows":` + leaf + `},"required":["rows"]}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("union leaf collection checked17 authorized", err)
		}
	}
}

func TestBorrowedCollectionUnionLeavesFullSourceCacheAndNominalRecursion(t *testing.T) {
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
			doc := schemas(`{"Node":{"oneOf":[{"properties":{"kind":{"const":"id"},"id":{"type":"string"},"children":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}},"required":["id"]},{"properties":{"kind":{"const":"label"},"label":{"type":"string"}},"required":["label"]}]},"Source":{"properties":{` + strings.Join(props, ",") + `}},` + root + `"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`)
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
		doc := schemas(`{"A":{"oneOf":[{"properties":{"kind":{"const":"values"},"values":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/rows"}}},"required":["values"]},{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]}]},"B":{"oneOf":[{"properties":{"kind":{"const":"a"},"a":{"$ref":"#/components/schemas/A"}},"required":["a"]},{"properties":{"kind":{"const":"id"},"id":{"type":"string"}},"required":["id"]}]},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/B"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		r := request(doc, "Dto")
		r.Options.Validation = checked
		if _, err := core.GenerateWithDiagnostics(r); err != nil {
			t.Fatal("actual nominal mutual recursion rejected", checked, err)
		}
	}
}
