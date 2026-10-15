package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestBorrowedCollectionEnumLeavesFirstHTTPUnionAndSlots(t *testing.T) {
	base := `"State":{"type":"string","enum":["ready","done"]},"Source":{"properties":{"rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/State"}}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/rows"}`
	for _, paths := range []string{
		`{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Bridge","nullable":false}}}}}}}}`,
		`{"/write":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Bridge"}}}},"responses":{"204":{}}}}}`,
		`{"/watch":{"get":{"responses":{"200":{"content":{"text/event-stream":{"schema":{"properties":{"event":{"const":"rows"},"data":{"$ref":"#/components/schemas/Bridge"}},"required":["data"]}}}}}}}}`,
	} {
		input, err := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": json.RawMessage(`{` + base + `}`)}, "paths": json.RawMessage(paths)})
		if err != nil {
			t.Fatal(err)
		}
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !strings.Contains(source(t, got, "DefaultApi"), "kotlin.collections.List<org.example.models.models.StateDto>") {
				t.Fatal("late canonical DTO leaf lost", checked, err)
			}
		}
	}
	input := schemas(`{` + base + `,"Choice":{"oneOf":[{"properties":{"kind":{"const":"rows"},"values":{"$ref":"#/components/schemas/Bridge","nullable":false}},"required":["values"]},{"properties":{"kind":{"const":"name"},"name":{"type":"string"}},"required":["name"]}]}}`)
	if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
		t.Fatal("union collection nominal leaf lost", err)
	}
	for _, count := range []int{240, 241} {
		var fields, names []string
		for i := 0; i < count; i++ {
			fields = append(fields, fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Bridge","nullable":false}`, i))
			names = append(names, fmt.Sprintf(`"p%d"`, i))
		}
		doc := schemas(`{` + base + `,"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(names, ",") + `]}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if count == 240 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0 {
			t.Fatal("nominal collection container lost one slot", count, err)
		}
	}
}
