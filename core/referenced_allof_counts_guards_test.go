package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedNestedAllOfCountsKeepExcludedContextsAndNullablePointers(t *testing.T) {
	count := `{"type":"object","allOf":[{"properties":{"v":{"type":"string"}},"minProperties":0}]}`
	for _, input := range [][]byte{
		schemas(`{"Choice":{"oneOf":[` + count + `,{"properties":{"other":{"type":"boolean"}}}]}}`),
		schemas(`{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind"}},"Leaf":{"allOf":[{"type":"object","allOf":[{"properties":{"kind":{"const":"leaf"},"v":{"type":"string"}},"minProperties":0}]}]}}`),
		fieldDoc(`{"properties":{"v":{"type":"string"}},"additionalProperties":` + count + `}`),
		httpDoc(`{"parameters":[{"name":"q","in":"query","schema":` + count + `}],"responses":{"204":{}}}`),
		httpDoc(`{"requestBody":{"content":{"application/json":{"schema":` + count + `}}},"responses":{"204":{}}}`),
	} {
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Excluded counts gained ordinary serializer privilege", string(input), e)
		}
	}
	for _, bounds := range []string{`"minProperties":0`, `"minProperties":-1`, `"maxProperties":"bad"`} {
		input := schemas(`{"Root":{"nullable":true,"type":"object","allOf":[{"type":"object","allOf":[{"properties":{"v":{"type":"string"}},` + bounds + `}]}]}}`)
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if bounds == `"minProperties":0` {
			if e != nil || !strings.Contains(source(t, got, "RootDto"), "element.size < 0") {
				t.Fatal("nullable nested allOf counts lost", e)
			}
			continue
		}
		key := strings.Split(bounds, `"`)[1]
		if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" || *e.(core.Diagnostics)[0].Pointer != "/components/schemas/Root/allOf/0/allOf/0/"+key || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("Nullable component count diagnostic changed", bounds, e)
		}
	}
	for _, n := range []int{100, 5500} {
		members := make([]string, n)
		for i := range members {
			members[i] = `{"$ref":"#/components/schemas/Source"}`
		}
		input := schemas(`{"Source":{"properties":{"value":{"type":"integer"}},"minProperties":0,"maxProperties":2},"Record":{"allOf":[` + strings.Join(members, ",") + `]}}`)
		r := request(input, "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (n == 100) || e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal(fmt.Sprint("Every repeated count source work ", n), e)
		}
		if e == nil && strings.Count(source(t, got, "RecordDto"), "element.size") != 2 {
			t.Fatal("Source-expanded count output")
		}
	}
}
