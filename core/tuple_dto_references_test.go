package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleDTOReferences(t *testing.T) {
	for _, tc := range []struct{ name, prefix, tail, want string }{
		{"same", `[{"$ref":"#/components/schemas/Item"},{"$ref":"#/components/schemas/Item"}]`, "", "org.example.tupledtos.models.ItemDto"},
		{"alias", `[{"$ref":"#/components/schemas/Alias","nullable":false},{"$ref":"#/components/schemas/Item"}]`, "", "org.example.tupledtos.models.ItemDto"},
		{"nullable", `[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Item","nullable":true}]`, "", "org.example.tupledtos.models.ItemDto?"},
		{"different", `[{"$ref":"#/components/schemas/Item"},{"$ref":"#/components/schemas/Other"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{"mixed", `[{"$ref":"#/components/schemas/Item"},{"type":"string"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{"tail", `[{"$ref":"#/components/schemas/Item"},{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Alias","nullable":false}`, "org.example.tupledtos.models.ItemDto"},
		{"scalar-tail", `[{"$ref":"#/components/schemas/Item"}]`, `,"items":{"type":"integer"}`, "kotlin.Int"},
		{"internal", `[{"$ref":"#/components/schemas/Owner/properties/node"},{"$ref":"#/components/schemas/Bridge"}]`, "", "org.example.tupledtos.models.OwnerDto.Node"},
		{"recursive", `[{"$ref":"#/components/schemas/Node"},{"$ref":"#/components/schemas/Node"}]`, "", "org.example.tupledtos.models.NodeDto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := schemas(`{"Item":{"properties":{"value":{"type":"string"},"count":{"type":"integer","default":7}},"required":["value"]},"Alias":{"$ref":"#/components/schemas/Item","nullable":true},"Other":{"properties":{"value":{"type":"string"},"count":{"type":"integer","default":7}},"required":["value"]},"Node":{"properties":{"value":{"type":"string"},"next":{"$ref":"#/components/schemas/Node"}},"required":["value"]},"Owner":{"properties":{"node":{"properties":{"name":{"type":"string"}},"required":["name"]}}},"Bridge":{"$ref":"#/components/schemas/Owner/properties/node"},"Probe":{"type":"array","prefixItems":` + tc.prefix + tc.tail + `}}`)
			original := bytes.Clone(input)
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					rq := request(input, "Dto")
					rq.Options.PackageName = "org.example.tupledtos"
					rq.Options.ModelStyle, rq.Options.Validation = style, checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
						t.Fatal("lost physical constructor/nullable identity", source(t, got, "ProbeDto"))
					}
					if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_TUPLE_APPROXIMATION" || *got.Warnings[0].Pointer != "/components/schemas/Probe/prefixItems" {
						t.Fatal("lost original approximation", got.Warnings)
					}
					again, err := core.GenerateWithDiagnostics(rq)
					if err != nil || !reflect.DeepEqual(got, again) || !bytes.Equal(input, original) {
						t.Fatal("source mutation or nondeterminism", err)
					}
				}
			}
		})
	}
}

func TestTupleDTOReferenceNoDropGuards(t *testing.T) {
	for _, tc := range []struct{ target, member, pointer string }{
		{`{"properties":{"value":{"type":"string"}}}`, `{"$ref":"#/components/schemas/Item","default":{}}`, "/components/schemas/Probe/prefixItems/0/default"},
		{`{"properties":{"value":{"type":"string"}}}`, `{"$ref":"#/components/schemas/Item","nullable":"bad"}`, "/components/schemas/Probe/prefixItems/0/nullable"},
		{`{"properties":{"value":{"type":"string"}},"additionalProperties":true}`, `{"$ref":"#/components/schemas/Item"}`, "/components/schemas/Item/additionalProperties"},
	} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Item":`+tc.target+`,"Probe":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				r, err := core.GenerateWithDiagnostics(rq)
				if err == nil || !reflect.DeepEqual(r, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
					t.Fatal("typed DTO source or own-role annotation hidden", tc, tail, checked, err)
				}
			}
		}
	}
	for _, target := range []string{`{}`, `{"properties":{"pin":{"const":"only"}}}`, `{"additionalProperties":{"type":"string"}}`, `{"oneOf":[{"type":"string"},{"type":"integer"}]}`} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			rq := request(schemas(`{"Item":`+target+`,"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}]`+tail+`}}`), "Dto")
			if r, err := core.GenerateWithDiagnostics(rq); err == nil || !reflect.DeepEqual(r, core.GenerationReport{}) {
				t.Fatal("unsupported nominal source acquired tuple permission", target, err)
			}
		}
	}
}

func TestTupleDTOUsesOriginalArrayCodecsAndDirectionalViews(t *testing.T) {
	input := schemas(`{"Item":{"properties":{"name":{"type":"string"},"id":{"type":"string","readOnly":true},"secret":{"type":"string","writeOnly":true}},"required":["name","id","secret"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Item"}},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"required":["rows"]}}`)
	input = bytes.Replace(input, []byte(`"components":`), []byte(`"paths":{"/echo":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Rows"}}}},"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Rows"}}}}}}}},"components":`), 1)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq := request(input, "Dto")
		rq.Options.ModelStyle = style
		before, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		rq.Document = bytes.Replace(input, []byte(`"items":{"$ref":"#/components/schemas/Item"}`), []byte(`"prefixItems":[{"$ref":"#/components/schemas/Item"}]`), 1)
		after, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(before.Artifacts, after.Artifacts) {
			t.Fatal("tuple ref replaced original array codecs or directional views", err)
		}
		if len(after.Warnings) != len(before.Warnings)+1 {
			t.Fatal("missing tuple-specific approximation", after.Warnings)
		}
	}
}

func TestTupleDTOOwnDefaultAndUniqueRolesRemainSeparate(t *testing.T) {
	for _, tc := range []struct{ probe, pointer string }{
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"default":[{"value":"valid"}]}`, "/components/schemas/Probe/default"},
		{`{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"default":[{"value":"valid"}]}}}`, "/components/schemas/Probe/properties/rows/prefixItems"},
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"uniqueItems":true}`, "/components/schemas/Probe/uniqueItems"},
		{`{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Item"}],"items":{"$ref":"#/components/schemas/Item"},"uniqueItems":true}`, "/components/schemas/Probe/uniqueItems"},
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Item":{"properties":{"value":{"type":"string"}},"required":["value"]},"Probe":`+tc.probe+`}`), "Dto")
			rq.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(rq)
			if checked && strings.Contains(tc.probe, `"items":{"$ref":"#/components/schemas/Item"}`) && strings.Contains(tc.probe, `"uniqueItems":true`) {
				if err != nil || len(got.Warnings) == 0 || !strings.Contains(source(t, got, "ProbeDto"), "List<org.example.models.models.ItemDto>") {
					t.Fatal("originalhomogeneousDTOtail uniqueness", err)
				}
				continue
			}
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
				t.Fatal("tuple introduced an independent default/nominal uniqueness role", tc, checked, err)
			}
		}
	}
}
