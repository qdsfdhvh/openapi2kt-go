package core_test

import (
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCollectionTemporalLeavesLateHTTPAndUnionUses(t *testing.T) {
	product := borrowedTemporalSource()
	catalog := `"Node":` + product + `,"Source":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}}`
	for _, tail := range []string{
		`,"paths":{"/echo":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Source/properties/rows"}}}},"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Source/properties/rows"}}}}}}}}`,
		`,"paths":{"/events":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"properties":{"event":{"const":"rows","type":"string"},"data":{"$ref":"#/components/schemas/Source/properties/rows"}},"required":["data"]}}}}}}}}`,
	} {
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + catalog + `}}` + tail + `}`)
		for _, checked := range []bool{false, true} {
			rq := request(doc, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("late first product collection use rejected", checked, err)
			}
			all := ""
			for _, file := range got.Artifacts.Files {
				all += file.Content
			}
			if !strings.Contains(all, "kotlin.collections.List<kotlin.time.Instant?>") {
				t.Fatal("late use lost actual product backing")
			}
		}
	}
	doc := schemas(`{` + catalog + `,"Choice":{"oneOf":[{"properties":{"tag":{"const":"rows"},"data":{"$ref":"#/components/schemas/Source/properties/rows"}},"required":["data"]},{"properties":{"tag":{"const":"id"},"id":{"type":"string"}},"required":["id"]}]}}`)
	if _, err := core.GenerateWithDiagnostics(request(doc, "Dto")); err != nil {
		t.Fatal("late union product collection use rejected", err)
	}
}
