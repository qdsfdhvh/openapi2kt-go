package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleStringEnumReferences(t *testing.T) {
	for _, tc := range []struct{ name, prefix, tail, want string }{
		{"same", `[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Choice"}]`, "", "org.example.tupleenums.models.ChoiceDto"},
		{"alias", `[{"$ref":"#/components/schemas/Alias","nullable":false},{"$ref":"#/components/schemas/Choice"}]`, "", "org.example.tupleenums.models.ChoiceDto"},
		{"nullable", `[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Choice","nullable":true}]`, "", "org.example.tupleenums.models.ChoiceDto?"},
		{"different", `[{"$ref":"#/components/schemas/Choice"},{"$ref":"#/components/schemas/Other"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{"mixed", `[{"$ref":"#/components/schemas/Choice"},{"type":"string"}]`, "", "kotlinx.serialization.json.JsonElement"},
		{"tail", `[{"$ref":"#/components/schemas/Choice"},{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Alias","nullable":false}`, "org.example.tupleenums.models.ChoiceDto"},
		{"scalar-tail", `[{"$ref":"#/components/schemas/Choice"}]`, `,"items":{"type":"integer"}`, "kotlin.Int"},
		{"internal", `[{"$ref":"#/components/schemas/Source/properties/mode"},{"$ref":"#/components/schemas/Bridge"}]`, "", "org.example.tupleenums.models.SourceDto.Mode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := schemas(`{"Choice":{"type":"string","enum":["ready","done"]},"Alias":{"$ref":"#/components/schemas/Choice","nullable":true},"Other":{"type":"string","enum":["ready","done"]},"Source":{"properties":{"mode":{"type":"string","enum":["on","off"]}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/mode"},"Probe":{"type":"array","prefixItems":` + tc.prefix + tc.tail + `}}`)
			original := bytes.Clone(input)
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				for _, checked := range []bool{false, true} {
					rq := request(input, "Dto")
					rq.Options.PackageName = "org.example.tupleenums"
					rq.Options.ModelStyle, rq.Options.Validation = style, checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(source(t, got, "ProbeDto"), "List<"+tc.want+">") {
						t.Fatal("lost actual vocabulary/nullable identity", source(t, got, "ProbeDto"))
					}
					if len(got.Warnings) != 1 || got.Warnings[0].Code != "W_TUPLE_APPROXIMATION" || *got.Warnings[0].Pointer != "/components/schemas/Probe/prefixItems" {
						t.Fatal("lost approximation", got.Warnings)
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

func TestTupleStringEnumFormerNoDropInputGraduates(t *testing.T) {
	// Exact former negative: the source is still validated even though items
	// selects Int. Its plain String vocabulary now has an actual owner.
	input := schemas(`{"Text":{"type":"string","enum":["a"]},"Probe":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Text"}],"items":{"type":"integer"}}}`)
	if got, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil || !strings.Contains(source(t, got, "ProbeDto"), "List<kotlin.Int>") {
		t.Fatal(err)
	}
}

func TestTupleStringEnumReferenceNoDropGuards(t *testing.T) {
	for _, tc := range []struct{ schema, member, pointer string }{
		{`{"type":"string","enum":["a"],"minLength":1}`, `{"$ref":"#/components/schemas/Choice"}`, "/components/schemas/Choice/minLength"},
		{`{"type":"string","enum":["a"]}`, `{"$ref":"#/components/schemas/Choice","nullable":"bad"}`, "/components/schemas/Probe/prefixItems/0/nullable"},
		{`{"type":"string","enum":["a"]}`, `{"$ref":"#/components/schemas/Choice","default":"a"}`, "/components/schemas/Probe/prefixItems/0/default"},
		{`{"type":"string","enum":[]}`, `{"$ref":"#/components/schemas/Choice"}`, "/components/schemas/Choice/enum"},
		{`{"type":"string","enum":["a",3]}`, `{"$ref":"#/components/schemas/Choice"}`, "/components/schemas/Choice/enum/1"},
	} {
		for _, tail := range []string{"", `,"items":{"type":"integer"}`} {
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Choice":`+tc.schema+`,"Probe":{"type":"array","prefixItems":[`+tc.member+`]`+tail+`}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) || *err.(core.Diagnostics)[0].Pointer != tc.pointer {
					t.Fatal("tuple hid original enum source", tc, tail, checked, err)
				}
			}
		}
	}
}
