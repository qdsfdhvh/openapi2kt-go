package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func nullableBuiltinChain(n int, terminal, consumer string) []byte {
	b := rootSpecialChain(n, terminal, consumer)
	return []byte(strings.Replace(string(b), `"A00":{"$ref":"#/components/schemas/Z"}`, `"A00":{"$ref":"#/components/schemas/Z","nullable":true}`, 1))
}
func TestRootNullablePrimitiveSourceBudget(t *testing.T) {
	for _, terminal := range []string{`{"type":"string"}`, `{"type":"integer","format":"int64"}`, `{"type":"number","format":"double"}`} {
		for _, flag := range []bool{false, true} {
			for _, typed := range []bool{false, true} {
				limit := 48
				if typed {
					limit = 46
				}
				for _, n := range []int{limit, limit + 1} {
					consumer := ""
					if typed {
						consumer = fmt.Sprintf(`{"properties":{"v":{"$ref":"#/components/schemas/A%02d","nullable":false}}}`, n-1)
					}
					req := request(nullableBuiltinChain(n, terminal, consumer), "Dto")
					req.Options.Validation = flag
					got, err := core.GenerateWithDiagnostics(req)
					if n == limit {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
						t.Fatal(n, got, err)
					}
				}
			}
		}
	}
}
func TestRootNullablePrimitiveLegacyPolicyAndCanonicalBounds(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, ann := range []string{"", `,"nullable":false`} {
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z"`+ann+`},"Z":{"type":"string"}}`), "Dto")
			req.Options.Validation = flag
			got, err := core.GenerateWithDiagnostics(req)
			if flag {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(source(t, got, "ADto"), "= kotlin.String\n") {
					t.Fatal(got)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" {
				t.Fatal("legacy default-off broadened", err)
			}
		}
		req := request(schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/Z"},"Z":{"type":"string"}}`), "Dto")
		req.Options.Validation = flag
		got, err := core.GenerateWithDiagnostics(req)
		if flag {
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(source(t, got, "ADto"), "= kotlin.String?") {
				t.Fatal(got)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Pointer == nil || *err.(core.Diagnostics)[0].Pointer != "/components/schemas/B" {
			t.Fatal("unsupported intermediate bypass", got, err)
		}
	}
	req := request(schemas(`{"A":{"$ref":"#/components/schemas/B","nullable":true},"B":{"$ref":"#/components/schemas/Z"},"Z":{"type":"string","minLength":2},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/A"},"n":{"$ref":"#/components/schemas/A","nullable":false}},"required":["n"]}}`), "Dto")
	req.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(req)
	if err != nil {
		t.Fatal(err)
	}
	if s := source(t, got, "HolderDto"); !strings.Contains(s, "isLowSurrogate") || !strings.Contains(s, "val n: kotlin.String,") {
		t.Fatal(s)
	}
}
func TestRootNullablePrimitiveGuards(t *testing.T) {
	for _, flag := range []bool{false, true} {
		for _, root := range []string{`{"type":"integer","const":1}`, `{"type":"boolean","enum":[true]}`, `{"type":"string","x-kotlin-value-class":false}`} {
			req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":`+root+`}`), "")
			req.Options.Validation = flag
			got, err := core.GenerateWithDiagnostics(req)
			if flag && strings.Contains(root, `"enum"`) {
				// Preserve the original own-nullable Boolean enum alias input.
				if err != nil || !strings.Contains(source(t, got, "A"), "typealias A = kotlin.Boolean?") || !strings.Contains(source(t, got, "Z"), "typealias Z = kotlin.Boolean\n") {
					t.Fatal(got, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
				t.Fatal(got, err)
			}
		}
	}
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"integer","format":"int64"}`
	aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/A"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
	aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/A"},{"type":"string"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
	aliasFailure(t, schemas(`{`+base+`,"Leaf":{"properties":{"id":{"type":"string"}}},"C":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"bad":"#/components/schemas/A"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/discriminator/mapping/bad")
}
