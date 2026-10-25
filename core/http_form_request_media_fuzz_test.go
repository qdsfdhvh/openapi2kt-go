package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzHTTPFormRequestMedia(f *testing.F) {
	f.Add("text", "application/x-www-form-urlencoded", uint8(0), true)
	f.Add("a/~+雪", "APPLICATION/X-WWW-FORM-URLENCODED; charset=UTF-8", uint8(1), false)
	f.Add("class", "application/json", uint8(2), true)
	f.Fuzz(func(t *testing.T, wire, media string, variant uint8, checked bool) {
		if len(wire) > 128 || len(media) > 128 {
			t.Skip()
		}
		leaf := map[string]any{"type": "string"}
		switch variant % 6 {
		case 1:
			leaf = map[string]any{"type": "boolean"}
		case 2:
			leaf = map[string]any{"type": "integer", "format": "int64"}
		case 3:
			leaf["nullable"] = true
		case 4:
			leaf["enum"] = []string{"a", "b"}
		case 5:
			leaf["default"] = "a"
		}
		d := map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/form": map[string]any{"post": map[string]any{"requestBody": map[string]any{"required": true, "content": map[string]any{media: map[string]any{"schema": map[string]any{"type": "object", "required": []string{wire}, "properties": map[string]any{wire: leaf}}}}}, "responses": map[string]any{"204": map[string]any{}}}}}}
		input, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		first, err1 := core.GenerateWithDiagnostics(rq)
		second, err2 := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err1, err2) {
			t.Fatal("form nondeterminism")
		}
		if err1 != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial form artifacts on error")
		}
	})
}
