package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func objectGraphDocument(member, value, extras string) []byte {
	return schemas(`{"Leaf":{"type":"string"},"Values":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Leaf"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values"}},"RowAlias":{"$ref":"#/components/schemas/Rows"},"Holder":{"properties":{"v":{"properties":{"member":` + member + `},"default":{"member":` + value + `}}}}` + extras + `}`)
}

func TestObjectDefaultGraphMemberCollections(t *testing.T) {
	for _, member := range []string{
		`{"$ref":"#/components/schemas/Rows"}`,
		`{"$ref":"#/components/schemas/RowAlias"}`,
		`{"type":"array","items":{"$ref":"#/components/schemas/Values"}}`,
		`{"$ref":"#/components/schemas/Hops/properties/rows"}`,
	} {
		input := objectGraphDocument(member, `[{"z":"last","a":"first"}]`, `,"Hops":{"properties":{"rows":{"$ref":"#/components/schemas/RowAlias"}}}`)
		for _, checked := range []bool{false, true} {
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			first, err := core.GenerateWithDiagnostics(rq)
			if err != nil {
				t.Fatal("finite member graph rejected", member, checked, err)
			}
			if !strings.Contains(source(t, first, "HolderDto"), `\"a\":\"first\",\"z\":\"last\"`) {
				t.Fatal("graph default lost original values/canonical key order", member)
			}
			second, err := core.GenerateWithDiagnostics(rq)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatal("graph source/output drift", err)
			}
		}
	}
}

func TestObjectDefaultGraphMemberValuePointers(t *testing.T) {
	for _, tc := range []struct{ value, tail string }{
		{`[{"a/~":false}]`, "/0/a~1~0"},
		{`null`, ""},
		{`[null]`, "/0"},
	} {
		input := objectGraphDocument(`{"$ref":"#/components/schemas/Rows"}`, tc.value, "")
		for _, required := range []bool{false, true} {
			if required {
				input = []byte(strings.Replace(string(input), `"Holder":{"properties"`, `"Holder":{"required":["v"],"properties"`, 1))
			}
			carrierDefaultFailure(t, input, false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tc.tail)
		}
	}
}

func TestObjectDefaultGraphMemberNearestNullability(t *testing.T) {
	for _, narrow := range []bool{false, true} {
		for _, tc := range []struct{ value, tail string }{
			{`null`, ""}, {`[null]`, "/0"}, {`[{"a":null}]`, "/0/a"},
		} {
			member := `{"$ref":"#/components/schemas/RowAlias"}`
			if narrow {
				member = `{"$ref":"#/components/schemas/RowAlias","nullable":false}`
			}
			input := objectGraphDocument(member, tc.value, "")
			input = []byte(strings.Replace(string(input), `"Rows":{"type":"array"`, `"Rows":{"nullable":true,"type":"array"`, 1))
			if tc.value == "null" && !narrow {
				if _, err := core.GenerateWithDiagnostics(request(input, "Dto")); err != nil {
					t.Fatal("nullable alias graph lost actual view", err)
				}
			} else {
				carrierDefaultFailure(t, input, false, "E_SCHEMA_INVALID", "/components/schemas/Holder/properties/v/default/member"+tc.tail)
			}
		}
	}
}

func TestObjectDefaultGraphSourceAnnotationsAndBudgets(t *testing.T) {
	base := string(objectGraphDocument(`{"$ref":"#/components/schemas/Rows"}`, `[]`, ""))
	for _, tc := range []struct{ old, replacement, pointer string }{
		{`"Rows":{"type":"array"`, `"Rows":{"default":[],"type":"array"`, "/components/schemas/Rows/default"},
		{`"RowAlias":{"$ref":"#/components/schemas/Rows"}`, `"RowAlias":{"$ref":"#/components/schemas/Rows","default":[]}`, "/components/schemas/RowAlias/default"},
		{`"items":{"$ref":"#/components/schemas/Values"}`, `"items":{"$ref":"#/components/schemas/Values","default":{}}`, "/components/schemas/Rows/items/default"},
	} {
		carrierDefaultFailure(t, []byte(strings.Replace(base, tc.old, tc.replacement, 1)), false, "E_SCHEMA_UNSUPPORTED", tc.pointer)
	}
	for _, width := range []int{240, 241} {
		fields := make([]string, width)
		for i := range fields {
			fields[i] = fmt.Sprintf(`"p%d":{"$ref":"#/components/schemas/Rows"}`, i)
		}
		input := schemas(`{"Values":{"additionalProperties":{"type":"string"}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values"}},"Holder":{"properties":{"v":{"properties":{` + strings.Join(fields, ",") + `},"default":{}}}}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if (err == nil) != (width == 240) || err != nil && (len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT") {
			t.Fatal("graph property slot budget changed", width, err)
		}
	}
}
