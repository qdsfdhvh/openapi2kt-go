package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestTupleBuiltinReferences(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[{"$ref":"#/components/schemas/Alias"},{"type":"string"}]`, "", "kotlin.String"},
		{`[{"$ref":"#/components/schemas/Date"},{"$ref":"#/components/schemas/Date"}]`, "", "kotlinx.datetime.LocalDate"},
		{`[{"$ref":"#/components/schemas/Alias","nullable":true},{"type":["string","null"]}]`, "", "kotlin.String?"},
		{`[{"$ref":"#/components/schemas/Alias"},{"type":"integer"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{`[{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Alias"}`, "kotlin.String"},
	} {
		input := schemas(`{"Text":{"type":"string","nullable":true},"Alias":{"$ref":"#/components/schemas/Text","nullable":false},"Date":{"type":"string","format":"date"},"Holder":{"properties":{"value":{"type":"array","prefixItems":` + tc.prefix + tc.tail + `}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err != nil {
			t.Fatal("finite scalar tuple ref rejected", tc, err, *err.(core.Diagnostics)[0].Pointer)
		}
		if !strings.Contains(source(t, got, "HolderDto"), "List<"+tc.want+">") {
			t.Fatal("lost actual scalar mapping", got)
		}
	}
}

func TestTupleReferenceNoDropOriginalSources(t *testing.T) {
	for _, tc := range []struct{ leaf, member, tail, pointer string }{
		{`{"type":"string","minLength":1}`, `{"$ref":"#/components/schemas/Text"}`, `,"items":{"type":"integer"}`, "/components/schemas/Text/minLength"},
		{`{"type":"string"}`, `{"$ref":"#/components/schemas/Text","nullable":"invalid"}`, `,"items":{"type":"integer"}`, "/components/schemas/Probe/prefixItems/0/nullable"},
		{`{"type":"string"}`, `{"$ref":"#/components/schemas/Text","default":"hidden"}`, `,"items":{"type":"integer"}`, "/components/schemas/Probe/prefixItems/0/default"},
	} {
		input := schemas(`{"Text":` + tc.leaf + `,"Probe":{"type":"array","prefixItems":[` + tc.member + `]` + tc.tail + `}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if err == nil || len(got.Artifacts.Files) != 0 || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("hidden or changed original source", tc, err)
			}
		}
	}
}
