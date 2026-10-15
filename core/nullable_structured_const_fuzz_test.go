package core_test

import (
	"reflect"
	"testing"

	"openapi-kt/core"
)

func FuzzNullableStructuredConst(f *testing.F) {
	for _, value := range []string{`{}`, `[]`, `{"$ref":"data","type":"data"}`, `[null,1e2]`, `{"n":1e400}`} {
		f.Add(value, false, false)
	}
	f.Fuzz(func(t *testing.T, value string, validation, narrow bool) {
		if len(value) > 4096 {
			t.Skip()
		}
		flag := "true"
		if narrow {
			flag = "false"
		}
		r := request(schemas(`{"Value":{"const":`+value+`,"nullable":true},"Alias":{"$ref":"#/components/schemas/Value","nullable":`+flag+`}}`), "Dto")
		r.Options.Validation = validation
		first, err := core.GenerateWithDiagnostics(r)
		second, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(err, again) || err != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Nullable literal/alias source drifted or produced partial files", err, again)
		}
	})
}
