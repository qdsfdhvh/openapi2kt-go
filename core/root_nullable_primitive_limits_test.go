package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestRootNullablePrimitiveSlots(t *testing.T) {
	for _, kind := range []string{"integer", "number"} {
		for _, narrow := range []bool{false, true} {
			for _, mode := range []int{0, 1, 2} {
				limit := 240
				if narrow {
					limit = 120
				}
				if mode == 2 {
					limit--
				}
				for _, n := range []int{limit, limit + 1} {
					fields, required, params := []string{}, []string{}, []string{}
					for i := range n {
						ref := `{"$ref":"#/components/schemas/A"}`
						if narrow {
							ref = `{"$ref":"#/components/schemas/A","nullable":false}`
						}
						fields = append(fields, fmt.Sprintf(`"p%d":%s`, i, ref))
						required = append(required, fmt.Sprintf(`"p%d"`, i))
						params = append(params, fmt.Sprintf(`{"name":"p%d","in":"query","required":true,"schema":%s}`, i, ref))
					}
					format := "int64"
					if kind == "number" {
						format = "double"
					}
					catalog := `"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"` + kind + `","format":"` + format + `"}`
					paths := ""
					if mode == 0 {
						catalog += `,"Holder":{"properties":{` + strings.Join(fields, ",") + `},"required":[` + strings.Join(required, ",") + `]}`
					} else {
						response := `{"204":{}}`
						if mode == 2 {
							response = `{"200":{"content":{"text/event-stream":{"schema":{"properties":{}}}}}}`
						}
						paths = `,"paths":{"/wide":{"get":{"parameters":[` + strings.Join(params, ",") + `],"responses":` + response + `}}}`
					}
					got, err := core.GenerateWithDiagnostics(request([]byte(`{"openapi":"3.1.0","components":{"schemas":{`+catalog+`}}`+paths+`}`), "Dto"))
					code := "E_MODEL_LIMIT"
					if mode != 0 {
						code = "E_OPERATION_LIMIT"
					}
					if n == limit {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || err.(core.Diagnostics)[0].Code != code || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
						t.Fatal(kind, narrow, mode, n, got, err)
					}
				}
			}
		}
	}
}
func TestRootNullablePrimitiveCheckedEdges(t *testing.T) {
	for _, n := range []int{16, 17} {
		leaf := `{"$ref":"#/components/schemas/A"}`
		for range n {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		req := request(schemas(`{"A":{"$ref":"#/components/schemas/Z","nullable":true},"Z":{"type":"string","minLength":2},"Holder":{"properties":{"v":`+leaf+`},"required":["v"]}}`), "Dto")
		req.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(req)
		if n == 16 {
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(source(t, got, "HolderDto"), "isLowSurrogate") {
				t.Fatal(got)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 || len(got.Warnings) != 0 {
			t.Fatal(got, err)
		}
	}
}
