package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestObjectCollectionRootFactoryControlAndOwnDefault(t *testing.T) {
	for _, tc := range []struct{ schema, literal string }{
		{`"type":"array","items":{"properties":{"value":{"type":"string"}},"required":["value"]}`, `[{"value":"ab"}]`},
		{`"type":"array","items":{"$ref":"#/components/schemas/T"}`, `[{"value":"ab"}]`},
		{`"type":"object","additionalProperties":{"properties":{"value":{"type":"string"}},"required":["value"]}`, `{"k":{"value":"ab"}}`},
	} {
		for _, own := range []bool{false, true} {
			annotation := ""
			if own {
				annotation = `,"default":` + tc.literal
			}
			for _, checked := range []bool{false, true} {
				rq := request(schemas(`{"Rows":{`+tc.schema+annotation+`},"T":{"properties":{"value":{"type":"string"}},"required":["value"]}}`), "Dto")
				rq.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal("accepted object collection lost factory role", own, checked, err)
				}
				if strings.Contains(source(t, got, "RowsDto"), "fun `RowsDto default`") != own {
					t.Fatal("root annotation/factory presence drift", own)
				}
			}
		}
	}
}

func TestObjectCollectionAllOfFactoryIsRetained(t *testing.T) {
	input := schemas(`{"Rows":{"type":"array","items":{"allOf":[{"properties":{"a":{"type":"string"}}},{"properties":{"b":{"type":"integer"}}}]},"default":[{"a":"ab","b":2}]}}`)
	got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
	if err != nil || !strings.Contains(source(t, got, "RowsDto"), "public fun `RowsDto default`") {
		t.Fatal("accepted allOf root annotation lost its factory", err)
	}
}

func TestObjectCollectionFactoriesKeepOtherLeafAndChildRoles(t *testing.T) {
	for _, leaf := range []string{
		`{"nullable":true,"oneOf":[{"type":"string"},{"type":"integer"}]}`,
		`{"properties":{"value":{"type":"string"}},"default":{}}`,
		`{"allOf":[{"properties":{"kind":{"const":"only"}}},{}]}`,
		`{"allOf":[{},{}]}`,
	} {
		input := schemas(`{"Rows":{"type":"array","items":` + leaf + `,"default":[]}}`)
		if got, err := core.GenerateWithDiagnostics(request(input, "Dto")); err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("unrelated leaf/child role acquired a factory", leaf, err)
		}
	}
}

func TestObjectCollectionFactoryValueBytesAndSlots(t *testing.T) {
	for _, size := range []int{32754, 32755} {
		input := schemas(`{"Rows":{"type":"array","items":{"properties":{"value":{"type":"string"}},"required":["value"]},"default":[{"value":"` + strings.Repeat("a", size) + `"}]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if size == 32754 && err != nil {
			t.Fatal("exact32KiB rejected", err)
		}
		if size > 32754 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("value byte budget bypassed", err)
		}
	}
	for _, count := range []int{240, 241} {
		properties := make([]string, count)
		for i := range properties {
			properties[i] = fmt.Sprintf(`"p%d":{"type":"string"}`, i)
		}
		input := schemas(`{"Rows":{"type":"array","items":{"properties":{` + strings.Join(properties, ",") + `}},"default":[{}]}}`)
		got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 240 && err != nil || count > 240 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || len(got.Artifacts.Files) != 0) {
			t.Fatal("constructor slots changed", count, err)
		}
	}
}

func TestObjectCollectionFactoryCannotSilentlyDropExcludedRootCounts(t *testing.T) {
	rq := request(schemas(`{"Rows":{"type":"object","nullable":true,"minProperties":0,"additionalProperties":{"properties":{"value":{"type":"string"}}},"default":null}}`), "Dto")
	rq.Options.Validation = true
	if got, err := core.GenerateWithDiagnostics(rq); err == nil || len(got.Artifacts.Files) != 0 {
		t.Fatal("excluded counted nullable root silently lost annotation", err)
	}
}
