package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNullableNativeEnumsKeepShapeExclusionsAndNoDrop(t *testing.T) {
	// Retain the exact former nullable unformatted factory refusal as positive.
	factory := request(schemas(`{"Code":{"type":"integer","nullable":true,"enum":[1],"default":null}}`), "Dto")
	factory.Options.Validation = true
	factoryOutput, factoryErr := core.GenerateWithDiagnostics(factory)
	if factoryErr != nil || !strings.Contains(source(t, factoryOutput, "CodeDto"), "public fun `CodeDto default`") {
		t.Fatal("nullable unformatted root factory lost", factoryErr)
	}
	base := `"Alias":{"$ref":"#/components/schemas/Code","nullable":false},"Code":{"type":["integer","null"],"enum":[1,2]}`
	for _, target := range []string{"Code", "Alias"} {
		for _, tail := range []string{`"Choice":{"allOf":[{"$ref":"#/components/schemas/` + target + `"},{"properties":{"id":{"type":"string"}}}]}`, `"Choice":{"oneOf":[{"$ref":"#/components/schemas/` + target + `"},{"type":"string"}]}`, `"Leaf":{"properties":{"id":{"type":"string"}}},"Choice":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"bad":"#/components/schemas/` + target + `"}}}`} {
			r := request(schemas(`{`+base+`,`+tail+`}`), "Dto")
			r.Options.Validation = true
			got, e := core.GenerateWithDiagnostics(r)
			if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(target, e)
			}
		}
	}
	for _, leaf := range []string{`{"type":["integer","null"],"enum":[1],"minimum":2}`, `{"type":["integer","null"],"enum":[null]}`, `{"type":"number","nullable":true,"enum":[1],"multipleOf":0.5}`} {
		r := request(schemas(`{"Code":`+leaf+`}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal(leaf, e)
		}
	}
	r := request(schemas(`{`+base+`,"Lost":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Code"}}},{"properties":{"value":{"type":"integer"}}}]}}`), "Dto")
	r.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(r)
	if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || !strings.HasSuffix(*e.(core.Diagnostics)[0].Pointer, "/enum") || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal(e)
	}
}

func TestNullableNativeEnumSourceAndCheckedDepthBudgets(t *testing.T) {
	for _, typed := range []bool{false, true} {
		limit := 48
		if typed {
			limit = 46
		}
		for _, n := range []int{limit, limit + 1} {
			consumer := ""
			if typed {
				consumer = fmt.Sprintf(`{"properties":{"value":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)
			}
			r := request(rootSpecialChain(n, `{"type":["integer","null"],"enum":[1,2]}`, consumer), "Dto")
			r.Options.Validation = true
			got, e := core.GenerateWithDiagnostics(r)
			if (e == nil) != (n == limit) {
				t.Fatal(typed, n, e)
			}
			if e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
				t.Fatal(e)
			}
		}
	}
	for _, depth := range []int{16, 17} {
		leaf := `{"$ref":"#/components/schemas/Code"}`
		for range depth {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		r := request(schemas(`{"Code":{"type":["integer","null"],"enum":[1,2]},"Holder":{"properties":{"values":`+leaf+`}}}`), "Dto")
		r.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(r)
		if (e == nil) != (depth == 16) {
			t.Fatal(depth, e)
		}
		if e != nil && (e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
			t.Fatal(e)
		}
	}
}
