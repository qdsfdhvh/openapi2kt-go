package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"testing"
)

func FuzzPrimitiveArrayDefaults(f *testing.F) {
	for _, value := range []string{`[]`, `[0,-1,1.25,null]`, `["$ref","quote\nline"]`, `[1e400]`, `{"type":"data"}`, `null`} {
		f.Add(value, false, false)
	}
	f.Fuzz(func(t *testing.T, value string, validation, nullable bool) {
		if len(value) > 4096 {
			t.Skip()
		}
		input := schemas(fmt.Sprintf(`{"Holder":{"properties":{"value":{"type":"array","nullable":%t,"items":{"type":"number","format":"float","nullable":%t},"default":%s}}}}`, nullable, nullable, value))
		r := request(input, "Dto")
		r.Options.Validation = validation
		first, err := core.GenerateWithDiagnostics(r)
		second, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, again) || err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Array default source drift or partial artifacts", err, again)
		}
	})
}
