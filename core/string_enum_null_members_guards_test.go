package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestStringEnumNullMembersRetainOriginalIndicesAndRejectedRoles(t *testing.T) {
	for _, tc := range []struct{ leaf, tail string }{
		{`{"type":["string","null"],"enum":["a",null,null]}`, "enum/2"},
		{`{"type":["string","null"],"enum":[null,"a","a"]}`, "enum/2"},
		{`{"type":["string","null"],"enum":[null,"a",1]}`, "enum/2"},
		{`{"type":["string","null"],"enum":[null]}`, "enum"},
		{`{"type":"string","enum":["a",null]}`, "enum/1"},
		{`{"type":["string","null"],"enum":["a",null],"format":"uuid"}`, "enum/1"},
		{`{"type":["string","null"],"enum":["a",null],"x-native-type":"bigint"}`, "enum/1"},
		{`{"type":["string","null"],"enum":["a",null],"default":"b"}`, "default"},
	} {
		for _, validation := range []bool{false, true} {
			r := request(fieldDoc(tc.leaf), "Dto")
			r.Options.Validation = validation
			got, e := core.GenerateWithDiagnostics(r)
			if e == nil || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/"+tc.tail) || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, validation, e)
			}
		}
	}
	for _, n := range []int{239, 240} {
		members := []string{"null"}
		for i := range n {
			members = append(members, fmt.Sprintf(`"value%d"`, i))
		}
		r := request(schemas(`{"State":{"type":["string","null"],"enum":[`+strings.Join(members, ",")+`]}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 239) || e != nil && (e.(core.Diagnostics)[0].Code != "E_ENUM_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal(n, e)
		}
	}
	input := fieldDoc(`{"anyOf":[{"type":["string","null"],"enum":["a",null]},{"type":"string"}]}`)
	failure(t, input, "E_SCHEMA_UNSUPPORTED", "/components/schemas/Record/properties/value/anyOf/0/type")
}

func TestStringEnumNullMembersKeepNonNullTagTokens(t *testing.T) {
	input := schemas(`{"Choice":{"oneOf":[{"properties":{"kind":{"type":["string","null"],"enum":[null,"a"]},"id":{"type":"integer"}},"required":["kind"]},{"properties":{"kind":{"type":["string","null"],"enum":["b",null]},"text":{"type":"string"}},"required":["kind"]}]}}`)
	got := report(t, input, "Dto")
	text := source(t, got, "ChoiceDto")
	if !strings.Contains(text, `tag == "a"`) || !strings.Contains(text, `tag == "b"`) || strings.Contains(text, `tag == "null"`) || strings.Contains(text, `serialName: kotlin.String = "null"`) {
		t.Fatal(text)
	}
	input = schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"fixed":"Leaf"}}},"Leaf":{"properties":{"kind":{"type":["string","null"],"enum":["fixed",null]}}}}`)
	got = report(t, input, "Dto")
	text = source(t, got, "LeafDto")
	if strings.Contains(text, "RootDtoKind?") || !strings.Contains(text, "public override val kind: org.example.models.models.RootDtoKind,") {
		t.Fatal(text)
	}
}
