package core_test

import (
	"fmt"
	"openapi-kt/core"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSchemaCollectionOwnedDTOEnvelope(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/schema-collection-owned-dtos.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []bool{false, true} {
		for _, suffix := range []string{"", "Dto"} {
			rq := request(input, suffix)
			rq.Options.Validation = flag
			r, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("collection-owned DTO", flag, suffix, err)
			}
			again, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(r, again) {
				t.Fatal("determinism", err)
			}
			owner := "org.example.models.models.NestedOwnerKey" + suffix
			root := "org.example.models.models.RootRows" + suffix + "Inner"
			value := "org.example.models.models.RootMap" + suffix + "Value"
			for _, want := range []string{"row: " + owner + ".RowsInner,", "root: " + root + ",", "nullableValue: " + value + "? = null", "narrowed: " + value + ",", "mixed: " + owner + ".BatchesInnerValueInner,", "child: " + owner + ".BatchesInnerValueInner.Child,", "List<" + root + ">", "Map<kotlin.String, " + owner + ".RowsInner>"} {
				if !strings.Contains(source(t, r, "Consumer"+suffix), want) {
					t.Fatal("canonical owner", want, source(t, r, "Consumer"+suffix))
				}
			}
			if !strings.Contains(source(t, r, "DefaultApi"), "input: "+root) || !strings.Contains(source(t, r, "Hops"+suffix), "next: "+root+"? = null") {
				t.Fatal("canonical HTTP/chain")
			}
			for _, want := range []string{"Patch<" + owner + ".EntriesValue>", "requiredNull: " + value + "?,", "requiredFalse: " + value + ","} {
				if !strings.Contains(source(t, r, "SubmitInput"+suffix), want) {
					t.Fatal("presence", want)
				}
			}
			for _, name := range []string{"RootRows" + suffix + "Inner", "RootMap" + suffix + "Value", "NestedOwnerKey" + suffix} {
				source(t, r, name)
			}
			if strings.Contains(source(t, r, "Consumer"+suffix), "public data class Row(") {
				t.Fatal("copied nominal")
			}
		}
	}
}

func TestSchemaCollectionOwnedDTORecursionLimitsAndGuards(t *testing.T) {
	report(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Rows/items")+`}},"Rows":{"type":"array","items":{"properties":{"self":`+pointerRef("Rows/items")+`,"other":`+pointerRef("Mapping/additionalProperties")+`}}},"Mapping":{"additionalProperties":{"properties":{"back":`+pointerRef("Rows/items")+`}}}}`), "")
	for _, n := range []int{240, 241} {
		var props []string
		for i := range n {
			props = append(props, fmt.Sprintf(`"p%d":%s`, i, pointerRef("Rows/items")))
		}
		doc := schemas(`{"Consumer":{"properties":{` + strings.Join(props, ",") + `}},"Rows":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}`)
		if n == 240 {
			aggregationReport(t, doc)
		} else {
			aliasFailure(t, doc, "E_MODEL_LIMIT", "/components/schemas/Consumer")
		}
	}
	for _, n := range []int{46, 47} {
		field := pointerRef("Rows/items")
		for range n {
			field = `{"type":"array","items":` + field + `}`
		}
		doc := schemas(`{"Consumer":{"properties":{"v":` + field + `}},"Rows":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}`)
		if n == 46 {
			report(t, doc, "")
		} else {
			failure(t, doc, "E_SCHEMA_DEPTH", "/components/schemas/Consumer/properties/v"+strings.Repeat("/items", n))
		}
	}
	for _, tc := range []struct{ owner, path string }{
		{`{"properties":{"fixed":{"type":"string"}},"additionalProperties":{"properties":{"id":{"type":"string"}}}}`, "Owner/additionalProperties"},
		{`{"type":"array","items":{"allOf":[{"properties":{"id":{"type":"string"}}}]}}`, "Owner/items"},
		{`{"type":"array","items":{"properties":{"id":{"type":"string","const":"fixed"}}}}`, "Owner/items"},
		{`{"additionalProperties":{"properties":{"id":{"$ref":"#/components/schemas/Literal"}}}}`, "Owner/additionalProperties"},
	} {
		input := schemas(`{"Consumer":{"properties":{"v":` + pointerRef(tc.path) + `}},"Literal":{"type":"string","const":"fixed"},"Owner":` + tc.owner + `}`)
		if strings.Contains(tc.owner, `"items":{"allOf"`) {
			for _, checked := range []bool{false, true} {
				req := request(input, "Dto")
				req.Options.Validation = checked
				r, e := core.GenerateWithDiagnostics(req)
				if e != nil || !strings.Contains(source(t, r, "ConsumerDto"), "v: org.example.models.models.OwnerDtoInner? = null") {
					t.Fatal("original allOf collection peer graduation", e)
				}
			}
			continue
		}
		aliasFailure(t, input, "E_REF_UNSUPPORTED", "/components/schemas/Consumer/properties/v")
	}
	// A real existing peer collision remains an ordinary naming error.
	failure(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Rows/items")+`}},"Rows":{"type":"array","items":{"properties":{"id":{"type":"string"}}}},"RowsInner":{"properties":{"id":{"type":"string"}}}}`), "E_NAME_COLLISION", "/components/schemas/Rows/items")
	// The unchanged root-item alias negative graduates under the new bounded
	// root/internal binding contract; other collection exclusions stay negative.
	for _, flag := range []bool{false, true} {
		rq := request(schemas(`{"Alias":`+pointerRef("Rows/items")+`,"Rows":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}`), "Dto")
		rq.Options.Validation = flag
		r, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, r, "AliasDto"), "public typealias AliasDto = org.example.models.models.RowsDtoInner") {
			t.Fatal("root-item alias canonical peer graduation", flag, err)
		}
	}
	// Original input graduates only to its actual ordinary DTO leaf.
	borrowedDTOGraduation(t, schemas(`{"Consumer":{"properties":{"v":`+pointerRef("Owner/properties/rows")+`}},"Owner":{"properties":{"rows":{"type":"array","items":`+pointerRef("Rows/items")+`}}},"Rows":{"type":"array","items":{"properties":{"id":{"type":"string"}}}}}`), "org.example.models.models.RowsDtoInner")
}

func TestSchemaCollectionOwnedDTOSourceChecksNoDrop(t *testing.T) {
	owner := `"Rows":{"type":"array","items":{"properties":{"id":{"type":"string","minLength":2}}}}`
	doc := schemas(`{"Consumer":{"properties":{"v":` + pointerRef("Rows/items") + `}},` + owner + `}`)
	r := aggregationReport(t, doc)
	if !strings.Contains(source(t, r, "RowsInner"), "length >= 2") {
		t.Fatal("source check missing")
	}
	failure(t, doc, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/items/properties/id/minLength")
	raw := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + owner + `}},"paths":{"/v":{"get":{"parameters":[{"in":"query","name":"v","schema":{"properties":{"node":` + pointerRef("Rows/items") + `}}}],"responses":{"204":{}}}}}}`)
	aliasFailure(t, raw, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/items/properties/id/minLength")
}
