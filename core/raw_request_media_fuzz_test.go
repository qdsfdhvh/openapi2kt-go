package core_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzRawRequestMedia(f *testing.F) {
	f.Add("text/plain", uint8(0), true)
	f.Add("TEXT/PLAIN; charset=ISO-8859-1", uint8(1), true)
	f.Add("application/octet-stream", uint8(2), false)
	f.Fuzz(func(t *testing.T, media string, kind uint8, required bool) {
		if len(media) > 128 {
			t.Skip()
		}
		s := map[string]any{"type": "string"}
		switch kind % 4 {
		case 1:
			s["nullable"] = true
		case 2:
			s["format"] = "binary"
		case 3:
			s["type"] = "integer"
		}
		d := map[string]any{"openapi": "3.1.0", "paths": map[string]any{"/raw": map[string]any{"post": map[string]any{"requestBody": map[string]any{"required": required, "content": map[string]any{media: map[string]any{"schema": s}}}, "responses": map[string]any{"204": map[string]any{}}}}}}
		input, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		original := bytes.Clone(input)
		rq := request(input, "Dto")
		first, err := core.GenerateWithDiagnostics(rq)
		second, retry := core.GenerateWithDiagnostics(rq)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, retry) || !bytes.Equal(input, original) {
			t.Fatal("raw request source mutation or nondeterminism")
		}
		if err != nil && len(first.Artifacts.Files) != 0 {
			t.Fatal("partial raw request error output")
		}
	})
}
