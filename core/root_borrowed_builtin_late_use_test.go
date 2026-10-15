package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestRootBorrowedBuiltinFirstHTTPAndUnionUses(t *testing.T) {
	for _, consumer := range []string{
		`"paths":{"/read":{"get":{"responses":{"200":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Source/properties/hop","nullable":false}}}}}}}}`,
		`"components":{"schemas":{"Source":{"properties":{"leaf":{"type":["integer","null"],"format":"int64"},"hop":{"$ref":"#/components/schemas/Bridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false},"Choice":{"oneOf":[{"properties":{"event":{"const":"a"},"value":{"$ref":"#/components/schemas/Source/properties/hop","nullable":false}},"required":["event","value"]},{"properties":{"event":{"const":"b"},"other":{"type":"boolean"}},"required":["event","other"]}]}}}`,
	} {
		input := `{"openapi":"3.1.0",` + consumer + `}`
		if strings.HasPrefix(consumer, `"paths"`) {
			input = `{"openapi":"3.1.0","components":{"schemas":{"Source":{"properties":{"leaf":{"type":["integer","null"],"format":"int64"},"hop":{"$ref":"#/components/schemas/Bridge","nullable":true}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf","nullable":false}}},` + consumer + `}`
		}
		for _, checked := range []bool{false, true} {
			r := request([]byte(input), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal("first shared use failed", checked, err)
			}
			if strings.HasPrefix(consumer, `"paths"`) {
				if !strings.Contains(source(t, got, "DefaultApi"), "): kotlin.Long") {
					t.Fatal("late HTTP borrowed root became nominal or nullable")
				}
			} else if !strings.Contains(source(t, got, "ChoiceDto"), "val value: kotlin.Long") {
				t.Fatal("first union source metadata probe lost borrowed role")
			}
		}
	}
}

func TestRootBorrowedBuiltinCheckedEdgesAndSourceRoles(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, leaf := range []string{`{"x-native-type":"bigint"}`, `{"type":"string","format":"byte"}`, `{"x-native-type":"kotlin.Long"}`, `true`} {
			r := request(schemas(`{"Source":{"properties":{"leaf":`+leaf+`}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_REF_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
				t.Fatal("borrowed builtin role expanded to special/JSON source", leaf, checked, err)
			}
		}
	}
	for _, count := range []int{16, 17} {
		leaf := `{"$ref":"#/components/schemas/Bridge","nullable":false}`
		for i := 0; i < count; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(schemas(`{"Source":{"properties":{"leaf":{"type":"string","minLength":2}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/leaf"},"Holder":{"properties":{"rows":`+leaf+`},"required":["rows"]}}`), "Dto")
		r.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(r)
		if count == 16 {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
			t.Fatal("borrowed alias created validation barrier", count, err)
		}
	}
}

func TestRootBorrowedBuiltinMixedCyclesAndNearestIntent(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, hop := range []string{`{"$ref":"#/components/schemas/Bridge"}`, `{"$ref":"#/components/schemas/Source/properties/hop"}`, `{"$ref":"#/components/schemas/Source/properties/leaf","default":"lost"}`} {
			r := request(schemas(`{"Source":{"properties":{"leaf":{"type":"string"},"hop":`+hop+`}},"Bridge":{"$ref":"#/components/schemas/Source/properties/hop"}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("cycle or semantic sibling accepted", checked, hop, err)
			}
		}
		for _, incoming := range []bool{false, true} {
			input := schemas(fmt.Sprintf(`{"Source":{"properties":{"leaf":{"type":["string","null"]},"after":{"$ref":"#/components/schemas/Source/properties/leaf","nullable":true},"before":{"$ref":"#/components/schemas/Bridge","nullable":%t}}},"Bridge":{"$ref":"#/components/schemas/Source/properties/after","nullable":false},"Root":{"$ref":"#/components/schemas/Source/properties/before"},"Holder":{"properties":{"root":{"$ref":"#/components/schemas/Root"},"false":{"$ref":"#/components/schemas/Root","nullable":false},"true":{"$ref":"#/components/schemas/Root","nullable":true}},"required":["root","false","true"]}}`, incoming))
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			suffix := ""
			if incoming {
				suffix = "?"
			}
			if !strings.Contains(source(t, got, "RootDto"), "= kotlin.String"+suffix+"\n") || !strings.Contains(source(t, got, "HolderDto"), "val `false`: kotlin.String,") || !strings.Contains(source(t, got, "HolderDto"), "val `true`: kotlin.String?") {
				t.Fatal("path-local override polluted shared suffix", source(t, got, "HolderDto"))
			}
		}
	}
}
