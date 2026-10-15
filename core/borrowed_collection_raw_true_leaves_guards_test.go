package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionRawTrueLeavesNullViewsAndNoDrop(t *testing.T) {
	product := borrowedRawTrueSource()
	doc := schemas(`{"Node":` + product + `,"Maybe":{"$ref":"#/components/schemas/Node","nullable":true},"False":{"$ref":"#/components/schemas/Maybe","nullable":false},"Source":{"properties":{"rows":{"type":["array","null"],"items":{"$ref":"#/components/schemas/Maybe"}},"map":{"additionalProperties":{"$ref":"#/components/schemas/False"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},"Narrow":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe","nullable":false}}},"required":["rows"]}}`)
	for _, checked := range []bool{false, true} {
		rq := request(doc, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "BridgeDto"), "List<kotlinx.serialization.json.JsonElement?>") || !strings.Contains(source(t, got, "NarrowDto"), "List<kotlinx.serialization.json.JsonElement>") {
			t.Fatal("product container and item intent merged")
		}
	}

}

func TestBorrowedCollectionRawTrueLeavesSourceAndCheckedBudgets(t *testing.T) {
	for _, hops := range []int{43, 44} {
		props := []string{`"terminal":{"$ref":"#/components/schemas/Node"}`}
		next := "terminal"
		for i := 0; i < hops; i++ {
			key := fmt.Sprintf("h%d", i)
			props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
			next = key
		}
		props = append(props, `"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
		doc := schemas(`{"Node":` + borrowedRawTrueSource() + `,"Source":{"properties":{` + strings.Join(props, ",") + `}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Source/properties/rows"}},"required":["value"]}}`)
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
		doc := schemas(`{"Node":` + borrowedRawTrueSource() + `,"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/Node"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"value":` + leaf + `},"required":["value"]}}`)
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

func borrowedRawTrueSource() string {
	return `{"type":"object","nullable":true}`
}

func TestBorrowedCollectionRawTrueLeavesRefGrammarNoDrop(t *testing.T) {
	for _, sibling := range []string{`"default":null`, `"pattern":"x"`, `"nullable":"invalid"`} {
		input := schemas(`{"Node":{"type":"object","nullable":true},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node",` + sibling + `}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("invalid raw leaf ref sibling discarded", sibling, err)
		}
	}
}
