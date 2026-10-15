package core_test

import (
	"fmt"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestHierarchyNullableAliasSourceDepthAndFalseConsumer(t *testing.T) {
	for _, typed := range []bool{false, true} {
		limit := 48
		if typed {
			limit = 46
		}
		for _, count := range []int{limit, limit + 1} {
			nodes := []string{`"Root":{"oneOf":[{"$ref":"#/components/schemas/First"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind"}}`, `"First":{"properties":{"kind":{"type":"string","const":"first"},"id":{"type":"string"}}}`, `"Second":{"properties":{"kind":{"type":"string","const":"second"},"id":{"type":"string"}}}`}
			for i := range count {
				target := "Root"
				intent := `,"nullable":true`
				if i > 0 {
					target = fmt.Sprint("A", i-1)
					intent = ""
				}
				nodes = append(nodes, fmt.Sprintf(`"A%d":{"$ref":"#/components/schemas/%s"%s}`, i, target, intent))
			}
			if typed {
				nodes = append(nodes, fmt.Sprintf(`"Holder":{"properties":{"narrowed":{"$ref":"#/components/schemas/A%d","nullable":false}}}`, count-1))
			}
			got, err := core.GenerateWithDiagnostics(request(schemas("{"+strings.Join(nodes, ",")+"}"), "Dto"))
			if count == limit {
				if err != nil {
					t.Fatal(typed, count, err)
				}
				alias := source(t, got, fmt.Sprint("A", count-1, "Dto"))
				if !strings.Contains(alias, "public typealias "+fmt.Sprint("A", count-1, "Dto")+" = org.example.models.models.RootDto?") {
					t.Fatal(alias)
				}
				if typed && !strings.Contains(source(t, got, "HolderDto"), "public val narrowed: org.example.models.models.RootDto? = null") {
					t.Fatal("optional presence default lost", got)
				}
			} else if err == nil || len(got.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
				t.Fatal(typed, count, got, err)
			}
		}
	}
	for _, tc := range []struct{ body, pointer string }{
		{`{"Alias":{"$ref":"#/components/schemas/Root","nullable":true},"Root":{"oneOf":[{"$ref":"#/components/schemas/First","nullable":true},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind"}},"First":{"properties":{"kind":{"type":"string","const":"first"}}},"Second":{"properties":{"kind":{"type":"string","const":"second"}}}}`, "/components/schemas/Root/oneOf/0/nullable"},
		{`{"Alias":{"$ref":"#/components/schemas/First","nullable":true},"Root":{"oneOf":[{"$ref":"#/components/schemas/Alias"},{"$ref":"#/components/schemas/Second"}],"discriminator":{"propertyName":"kind","mapping":{"first":"Alias","second":"Second"}}},"First":{"properties":{"kind":{"type":"string","const":"first"}}},"Second":{"properties":{"kind":{"type":"string","const":"second"}}}}`, "/components/schemas/Root/discriminator/mapping/first"},
	} {
		// Preserve both original member/mapping inputs: nullable aliases now
		// select the same non-null physical child codec at hierarchy edges.
		r := report(t, schemas(tc.body), "Dto")
		if !strings.Contains(source(t, r, "RootDto"), "org.example.models.models.FirstDto.serializer()") || !strings.Contains(source(t, r, "AliasDto"), "Dto?\n") {
			t.Fatal(tc.pointer, "nullable member or mapping identity")
		}
	}
}
