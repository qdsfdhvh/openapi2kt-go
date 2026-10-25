package core_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"openapi-kt/core"
	"openapi-kt/internal/document"
)

func FuzzPinnedStructuredConst(f *testing.F) {
	for _, literal := range []string{`{}`, `[]`, `{"$ref":"data","const":[null]}`, `[null,{"type":"data"}]`, `{"n":1e400}`} {
		f.Add(literal, false, false)
	}
	f.Fuzz(func(t *testing.T, literal string, checked, plain bool) {
		if len(literal) > 4096 {
			t.Skip()
		}
		r := request(schemas(`{"Choice":{"oneOf":[{"properties":{"tag":{"const":"one"},"payload":{"const":`+literal+`}}},{"properties":{"tag":{"const":"two"},"payload":{"const":{}}}}]}}`), "Dto")
		r.Options.Validation = checked
		if plain {
			r.Options.ModelStyle = core.PlainClass
		}
		first, e := core.GenerateWithDiagnostics(r)
		second, again := core.GenerateWithDiagnostics(r)
		if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(e, again) || e != nil && !reflect.DeepEqual(first, core.GenerationReport{}) {
			t.Fatal("Pinned const nondeterministic or partial", e, again)
		}
		var value any
		// encoding/json accepts duplicate keys; the document grammar does not.
		_, parseError := document.Load(r.Document, document.JSON, "")
		if parseError == nil && json.Unmarshal([]byte(literal), &value) == nil {
			switch value.(type) {
			case map[string]any, []any:
				if e != nil {
					t.Fatal("Valid structured body rejected", e)
				}
			}
		}
	})
}
