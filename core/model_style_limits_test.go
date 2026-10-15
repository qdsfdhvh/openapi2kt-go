package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestPlainStylePreservesIndependentAnalysisBudgets(t *testing.T) {
	var cases []struct {
		name string
		data []byte
		code string
	}
	for _, wide := range []bool{false, true} {
		max := 240
		nullable := true
		if wide {
			max = 120
			nullable = false
		}
		for _, n := range []int{max, max + 1} {
			props := map[string]any{}
			required := []string{}
			for i := 0; i < n; i++ {
				name := fmt.Sprintf("f%03d", i)
				props[name] = map[string]any{"type": "integer", "format": "int64", "nullable": nullable}
				required = append(required, name)
			}
			node := map[string]any{"Wide": map[string]any{"type": "object", "properties": props, "required": required}}
			data, _ := json.Marshal(node)
			code := ""
			if n > max {
				code = "E_MODEL_LIMIT"
			}
			cases = append(cases, struct {
				name string
				data []byte
				code string
			}{fmt.Sprintf("wide=%t/count=%d", wide, n), schemas(string(data)), code})
		}
	}
	for _, n := range []int{48, 49} {
		code := ""
		if n == 49 {
			code = "E_SCHEMA_DEPTH"
		}
		cases = append(cases, struct {
			name string
			data []byte
			code string
		}{fmt.Sprintf("source=%d", n), rootObjectChain(n, ""), code})
	}
	for _, n := range []int{16, 17} {
		leaf := `{"type":"string","minLength":1}`
		for i := 0; i < n; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		code := ""
		if n == 17 {
			code = "E_SCHEMA_DEPTH"
		}
		cases = append(cases, struct {
			name string
			data []byte
			code string
		}{fmt.Sprintf("checked=%d", n), schemas(`{"Holder":{"properties":{"values":` + leaf + `}}}`), code})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := request(tc.data, "Dto")
			req.Options.Validation = true
			base, baseErr := core.GenerateWithDiagnostics(req)
			req.Options.ModelStyle = core.PlainClass
			plain, plainErr := core.GenerateWithDiagnostics(req)
			if tc.code == "" {
				if baseErr != nil || plainErr != nil || len(plain.Artifacts.Files) != len(base.Artifacts.Files) {
					t.Fatal(baseErr, plainErr)
				}
				if strings.HasPrefix(tc.name, "wide=") && strings.Contains(source(t, plain, "WideDto"), "data class") {
					t.Fatal("width probe was not plain")
				}
			} else {
				if baseErr == nil || plainErr == nil || baseErr.(core.Diagnostics)[0].Code != tc.code || !reflect.DeepEqual(baseErr, plainErr) || len(plain.Artifacts.Files) != 0 || len(plain.Warnings) != 0 {
					t.Fatal(tc.name, baseErr, plainErr, plain)
				}
			}
		})
	}
}
