package core_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionEnumLeavesSourceCacheAndCheckedBudgets(t *testing.T) {
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
			doc := schemas(`{"State":{"type":["string","null"],"enum":["ready","done"]},"Source":{"properties":{` + strings.Join(props, ",") + `}},` + root + `"Holder":{"properties":{"value":` + field + `},"required":["value"]}}`)
			got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
			if hops <= 42 || hops == 43 && !cached {
				if err != nil {
					t.Fatal("original source48 rejected", cached, hops, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("enum erased preceding source work", cached, hops, err)
			}
		}
	}
	for _, depth := range []int{15, 16} {
		leaf := `{"$ref":"#/components/schemas/Bridge"}`
		for i := 0; i < depth; i++ {
			leaf = `{"type":"array","minItems":1,"items":` + leaf + `}`
		}
		doc := schemas(`{"State":{"type":"string","enum":["ready"]},"Source":{"properties":{"rows":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/State"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"Holder":{"properties":{"rows":` + leaf + `},"required":["rows"]}}`)
		rq := request(doc, "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if depth == 15 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("enum collection checked17 authorized", err)
		}
	}
}

func TestBorrowedCollectionEnumLeavesVocabularyGrammarAndExcludedRoles(t *testing.T) {
	for _, count := range []int{240, 241} {
		values := []string{}
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("state%d", i))
		}
		b, _ := json.Marshal(values)
		doc := schemas(`{"State":{"type":"string","enum":` + string(b) + `},"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/State"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("vocabulary limit lost", err)
		}
	}
	for _, node := range []string{`{"type":"string","enum":["ready",null]}`, `{"type":"string","enum":["ready","ready"]}`, `{"type":"string","enum":["ready"],"default":"ready"}`, `{"type":"integer","enum":[1,2]}`, `{"type":"string","enum":["ready"],"x-native-type":"org.example.State"}`, `{"type":"string","enum":["ready"],"format":"date"}`, `{"type":"string","x-kotlin-value-class":true}`, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`} {
		doc := schemas(`{"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/State"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"},"State":` + node + `}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if node == `{"oneOf":[{"type":"string"},{"type":"integer"}]}` {
			borrowedUnionGraduation(t, doc, "org.example.models.models.StateDto")
			continue
		}
		if node == `{"type":"string","x-kotlin-value-class":true}` {
			// Keep the historical input as the independent named-wrapper profile positive.
			if err != nil || !strings.Contains(source(t, got, "BridgeDto"), "kotlin.collections.List<org.example.models.models.StateDto>") {
				t.Fatal("original wrapper leaf graduation lost canonical physical type", err)
			}
			continue
		}
		if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("excluded vocabulary/source role authorized", node, err)
		}
	}
}
