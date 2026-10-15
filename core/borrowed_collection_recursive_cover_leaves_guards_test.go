package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionRecursiveCoverLeavesIncomingSCCAndCheckedBudgets(t *testing.T) {
	for _, tc := range []struct {
		catalog string
		height  int
	}{
		{`"Node":` + coverChainRoot("Node", "Node"), 2},
		{`"Node":` + coverChainRoot("ToOther", "Other") + `,"ToOther":{"$ref":"#/components/schemas/Other"},"Other":` + coverChainRoot("ToNode", "Node") + `,"ToNode":{"$ref":"#/components/schemas/Node"}`, 6},
	} {
		for _, cached := range []bool{false, true} {
			limit := 43 - tc.height
			if cached {
				limit--
			}
			for _, hops := range []int{limit, limit + 1} {
				props := []string{`"terminal":{"$ref":"#/components/schemas/Node"}`}
				next := "terminal"
				for i := 0; i < hops; i++ {
					key := fmt.Sprintf("h%d", i)
					props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
					next = key
				}
				props = append(props, `"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
				use := `{"$ref":"#/components/schemas/Source/properties/rows","nullable":false}`
				bridge := ""
				if cached {
					bridge = `,"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false}`
					use = `{"$ref":"#/components/schemas/Bridge","nullable":false}`
				}
				input := schemas(`{` + tc.catalog + `,"Source":{"properties":{` + strings.Join(props, ",") + `}}` + bridge + `,"Holder":{"properties":{"rows":` + use + `},"required":["rows"]}}`)
				got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
				if hops == limit {
					if err != nil {
						t.Fatal("completegraph source48 rejected", tc.height, cached, hops, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("SCCsuffix/cachefalse erased source49", tc.height, cached, hops, err)
				}
			}
		}
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		input := schemas(`{"Node":` + coverChainRoot("Node", "Node") + `,"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"rows":` + leaf + `},"required":["rows"]}}`)
		rq := request(input, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("cover erased checked17", err)
		}
	}
}

func TestBorrowedCollectionRecursiveCoverLeavesGrammarAndNullViews(t *testing.T) {
	for _, terminal := range []string{strings.Replace(coverChainRoot("Node", "Node"), `{"type":"string"}`, `{"type":"string","minLength":1}`, 1), strings.Replace(coverChainRoot("Node", "Node"), `{"type":"boolean"}`, `{"type":"string"}`, 1)} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Node":`+terminal+`,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`), "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed cover discarded source grammar", err)
		}
	}
	input := schemas(`{"Node":` + coverChainRoot("Node", "Node") + `,"Maybe":{"$ref":"#/components/schemas/Node","nullable":true},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":["array","null"],"items":{"$ref":"#/components/schemas/Maybe"}},"narrow":{"type":"array","items":{"$ref":"#/components/schemas/False"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},"Narrow":{"$ref":"#/components/schemas/Source/properties/narrow"}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "BridgeDto"), "List<kotlinx.serialization.json.JsonElement?>") || !strings.Contains(source(t, got, "NarrowDto"), "List<kotlinx.serialization.json.JsonElement>") {
		t.Fatal("cover container/item nullable merged")
	}
}
