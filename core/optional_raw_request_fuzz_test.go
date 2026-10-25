package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzOptionalRawRequestReferencedPresence(f *testing.F) {
	f.Add(uint8(0), false, false, false)
	f.Add(uint8(3), true, true, true)
	f.Add(uint8(87), false, false, false)
	f.Fuzz(func(t *testing.T, hops uint8, bytes, nullable, checked bool) {
		media := "text/plain"
		leaf := map[string]any{"type": "string"}
		if bytes {
			media = "application/octet-stream"
			leaf["format"] = "binary"
		}
		if nullable {
			leaf["nullable"] = true
		}
		components := map[string]any{"Leaf": leaf}
		previous := "Leaf"
		for i := 0; i < int(hops%5); i++ {
			name := fmt.Sprintf("Alias%d", i)
			components[name] = map[string]any{"$ref": "#/components/schemas/" + previous}
			previous = name
		}
		doc := map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": components}, "paths": map[string]any{"/raw": map[string]any{"post": map[string]any{"requestBody": map[string]any{"content": map[string]any{media: map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + previous}}}}, "responses": map[string]any{"204": map[string]any{}}}}}}
		input, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err := core.GenerateWithDiagnostics(rq)
		// Optional presence must not expand the shared resolver's existing
		// source admission. Compare the same source under required=true.
		doc["paths"].(map[string]any)["/raw"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["required"] = true
		control, marshalErr := json.Marshal(doc)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		required := rq
		required.Document = control
		_, sourceErr := core.GenerateWithDiagnostics(required)
		if sourceErr != nil {
			if len(first.Artifacts.Files) != 0 || !reflect.DeepEqual(err, sourceErr) {
				t.Fatal("optional presence changed original source/nullability rejection", err, sourceErr)
			}
			return
		}
		if nullable || err != nil {
			t.Fatal("optional raw source admission mismatch", nullable, err)
		}
		second, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(first, second) {
			t.Fatal("optional raw nondeterminism", err)
		}
	})
}
