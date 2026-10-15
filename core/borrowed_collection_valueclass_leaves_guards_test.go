package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionValueClassLeavesGrammarAndNominalBoundaries(t *testing.T) {
	for _, node := range []string{`{"type":"string","x-kotlin-value-class":false}`, `{"type":"string","x-kotlin-value-class":true,"default":"ok"}`, `{"type":"string","x-kotlin-value-class":true,"enum":["ok"]}`, `{"type":"string","x-kotlin-value-class":true,"format":"date"}`, `{"type":"string","x-native-type":"org.example.Value"}`, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`} {
		doc := schemas(`{"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Value"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Value":` + node + `}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if node == `{"oneOf":[{"type":"string"},{"type":"integer"}]}` {
			borrowedUnionGraduation(t, doc, "org.example.models.models.ValueDto")
			continue
		}
		if node == `{"type":"string","x-native-type":"org.example.Value"}` {
			// Preserve the original input as an explicit caller-native leaf positive.
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "List<org.example.Value>") {
				t.Fatal("caller-owned source lost its actual provider backing", err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded named-wrapper source role authorized", node, err)
		}
	}
	inline := schemas(`{"Source":{"properties":{"rows":{"type":"array","items":{"type":"string","x-kotlin-value-class":true}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
	if got, err := core.GenerateWithDiagnostics(request(inline, "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("borrowed inline wrapper acquired named ownership", err)
	}
}

func TestBorrowedCollectionValueClassLeavesSourceCacheAndCheckedBudgets(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, hops := range []int{42, 43, 44} {
			props := []string{`"terminal":{"$ref":"#/components/schemas/State"}`}
			next := "terminal"
			for i := 0; i < hops; i++ {
				key := fmt.Sprintf("h%d", i)
				props = append(props, fmt.Sprintf(`%q:{"$ref":"#/components/schemas/Source/properties/%s","nullable":false}`, key, next))
				next = key
			}
			props = append(props, `"rows":{"type":"array","items":{"$ref":"#/components/schemas/Source/properties/`+next+`","nullable":false}}`)
			field, root := `{"$ref":"#/components/schemas/Source/properties/rows"}`, ""
			if cached {
				root = `"Bridge":{"$ref":"#/components/schemas/Source/properties/rows","nullable":false},`
				field = `{"$ref":"#/components/schemas/Bridge","nullable":false}`
			}
			doc := schemas(`{"State":{"type":["integer","null"],"format":"int64","x-kotlin-value-class":true,"minimum":2},"Source":{"properties":{` + strings.Join(props, ",") + `}},` + root + `"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`)
			got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
			if hops <= 42 || hops == 43 && !cached {
				if err != nil {
					t.Fatal("original source48 rejected", cached, hops, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("wrapper erased preceding source work", cached, hops, err)
			}
		}
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		doc := schemas(`{"State":{"type":"integer","format":"int64","x-kotlin-value-class":true,"minimum":2},"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/State"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"rows":` + leaf + `},"required":["rows"]}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("wrapper collection checked17 authorized", err)
		}
	}
}
