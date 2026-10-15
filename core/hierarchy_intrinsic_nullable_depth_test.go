package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestIntrinsicHierarchyNullableAliasDepthCacheAndFalseNarrowing(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, typed := range []bool{false, true} {
			limit := 48
			if typed {
				limit = 46
			}
			for _, count := range []int{limit, limit + 1} {
				nodes := []string{`"Root":{"nullable":true,"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":{"first":"First","second":"Second"}}}`, `"First":{"nullable":true,"properties":{"kind":{"type":"string","const":"first"},"id":{"type":"string"}}}`, `"Second":{"properties":{"kind":{"type":"string","const":"second"},"id":{"type":"string"}}}`}
				for i := range count {
					target := "Root"
					if i > 0 {
						target = fmt.Sprint("A", i-1)
					}
					nodes = append(nodes, fmt.Sprintf(`"A%d":{"$ref":"#/components/schemas/%s"}`, i, target))
				}
				if typed {
					nodes = append(nodes, fmt.Sprintf(`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/A%d","nullable":false}},"required":["value"]}`, count-1))
				}
				req := request(schemas("{"+strings.Join(nodes, ",")+"}"), "Dto")
				req.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(req)
				if count == limit {
					if err != nil {
						t.Fatal(checked, typed, count, err)
					}
					if !strings.Contains(source(t, got, fmt.Sprint("A", count-1, "Dto")), "= org.example.models.models.RootDto?") {
						t.Fatal("intrinsic intent lost")
					}
					if typed && !strings.Contains(source(t, got, "HolderDto"), "public val value: org.example.models.models.RootDto,") {
						t.Fatal("false doesn't narrow")
					}
				} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
					t.Fatal(checked, typed, count, got, err)
				}
			}
		}
	}
}
