package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestIntrinsicNullablePrimitiveSourceBudgets(t *testing.T) {
	for _, terminal := range []string{`{"type":"string","nullable":true}`, `{"type":["integer","null"],"format":"int64"}`, `{"type":"number","format":"double","nullable":true}`} {
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
					req := request(rootSpecialChain(n, terminal, consumer), "Dto")
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
func TestIntrinsicNullablePrimitiveSlots(t *testing.T) {
	for _, scalar := range []string{"integer", "number"} {
		for _, narrow := range []bool{false, true} {
			for _, stream := range []int{0, 1, 2} {
				limit := 240
				if narrow {
					limit = 120
				}
				if stream == 2 {
					limit--
					if !narrow { /* Flow one extra slot */
					}
				}
				for _, n := range []int{limit, limit + 1} {
					var fields, required, params []string
					for i := range n {
						ref := `{"$ref":"#/components/schemas/A"}`
						if narrow {
							ref = `{"$ref":"#/components/schemas/A","nullable":false}`
						}
						fields = append(fields, fmt.Sprintf(`"p%d":%s`, i, ref))
						required = append(required, fmt.Sprintf(`"p%d"`, i))
						params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":%s}`, i, ref))
					}
					catalog := `"A":{"$ref":"#/components/schemas/Z"},"Z":{"type":"` + scalar + `","nullable":true`
					if scalar == "integer" {
						catalog += `,"format":"int64"`
					} else {
						catalog += `,"format":"double"`
					}
					catalog += `}`
					paths := ""
					if stream == 0 {
						catalog += `,"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
					} else {
						responses := `{"204":{}}`
						if stream == 2 {
							responses = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
						}
						paths = `,"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + responses + `}}}`
					}
					input := []byte(`{"openapi":"3.1.0","components":{"schemas":{` + catalog + `}}` + paths + `}`)
					got, err := core.GenerateWithDiagnostics(request(input, "Dto"))
					code := "E_MODEL_LIMIT"
					if stream != 0 {
						code = "E_OPERATION_LIMIT"
					}
					if n == limit {
						if err != nil {
							t.Fatal(n, scalar, narrow, stream, err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != code || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
						t.Fatal(n, got, err)
					}
				}
			}
		}
	}
}
func TestIntrinsicNullablePrimitiveNoDrop(t *testing.T) {
	base := `"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":{"type":"integer","format":"int64","nullable":true}`
	for _, target := range []string{"A", "Z"} {
		aliasFailure(t, schemas(`{`+base+`,"C":{"allOf":[{"$ref":"#/components/schemas/`+target+`"},{"properties":{"id":{"type":"string"}}}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/allOf/0/$ref")
		aliasFailure(t, schemas(`{`+base+`,"C":{"oneOf":[{"$ref":"#/components/schemas/`+target+`"},{"type":"string"}]}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/oneOf/0")
		aliasFailure(t, schemas(`{`+base+`,"Leaf":{"properties":{"id":{"type":"string"}}},"C":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"tag","mapping":{"bad":"#/components/schemas/`+target+`"}}}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/C/discriminator/mapping/bad")
	}
	for _, flag := range []bool{false, true} {
		// Nonnull scalar aliases remain validation-opt-in. Own-null refs to
		// NONNULL builtins graduate under the separate root binding contract.
		for _, input := range []string{`{"Z":{"type":"string","nullable":true,"const":"x"}}`, `{"Z":{"type":"boolean","nullable":true,"enum":[true]}}`} {
			req := request(schemas(input), "")
			req.Options.Validation = flag
			got, err := core.GenerateWithDiagnostics(req)
			if flag && strings.Contains(input, `"enum"`) {
				// Keep the original Boolean enum input as an explicit checked
				// nullable-root graduation; default-off remains rejected.
				if err != nil || !strings.Contains(source(t, got, "Z"), "typealias Z = kotlin.Boolean?") {
					t.Fatal(got, err)
				}
				continue
			}
			if err == nil || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
				t.Fatal(got, err)
			}
		}
	}
}
func TestIntrinsicNullablePrimitiveBounds(t *testing.T) {
	for _, flag := range []bool{false, true} {
		req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":false},"Z":{"type":["string","null"],"minLength":2,"maxLength":4},"Holder":{"required":["a","b"],"properties":{"a":{"$ref":"#/components/schemas/A"},"b":{"type":"array","items":{"$ref":"#/components/schemas/Z"}}}}}`), "Dto")
		req.Options.Validation = flag
		got, err := core.GenerateWithDiagnostics(req)
		if !flag {
			if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_UNSUPPORTED" || len(got.Artifacts.Files) != 0 {
				t.Fatal(got, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		s := source(t, got, "HolderDto")
		if !strings.Contains(s, "require(") || !strings.Contains(s, "isLowSurrogate") {
			t.Fatal("lost validation", s)
		}
	}
}
