package cli_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestCarrierDefaultGuardsPreserveOutput(t *testing.T) {
	oversize := `"` + strings.Repeat("a", 32767) + `"`
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		put(t, p.Input, rawJSONFieldDocument())
		args := rawJSONFieldArgs(p.Input, p.Output, checked, core.DataClass)
		result(t, invoke(t, p.Root, args...), 0)
		before := snapshot(t, p.Output)
		for _, tc := range []struct{ schemas, code, pointer string }{
			{`{"Holder":{"required":["open"],"properties":{"open":{"type":"object","additionalProperties":true,"default":null}}}}`, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/open/default"},
			{`{"Holder":{"properties":{"free":{"default":{"z/~":` + oversize + `,"a":0}}}}}`, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/free/default/z~1~0"},
			{`{"Holder":{"properties":{"u":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":` + oversize + `}}}}`, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/u/default"},
			{`{"V":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Holder":{"properties":{"u":{"$ref":"#/components/schemas/V","default":` + oversize + `}}}}`, "E_MODEL_LIMIT", "/components/schemas/Holder/properties/u/default"},
			{`{"Holder":{"allOf":[{"properties":{"u":{"oneOf":[{"type":"string"},{"type":"integer"}],"default":` + oversize + `}}},{"properties":{"u":{"type":"string"}}}]}}`, "E_MODEL_LIMIT", "/components/schemas/Holder/allOf/0/properties/u/default"},
		} {
			put(t, p.Input, []byte(`{"openapi":"3.1.0","components":{"schemas":`+tc.schemas+`}}`))
			failure := fails(t, invoke(t, p.Root, args...), 1, tc.code)
			if len(failure.Files) != 0 || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer {
				t.Fatalf("expected %s at %s: %+v", tc.code, tc.pointer, failure)
			}
			if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("default rejection changed existing files/modes/mtimes")
			}
		}
	}
}
