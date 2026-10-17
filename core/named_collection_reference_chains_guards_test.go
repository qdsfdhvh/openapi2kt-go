package core_test

import (
	"fmt"
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestNamedCollectionReferenceChainsFullSourceAndCheckedHeight(t *testing.T) {
	for _, terminal := range []string{`{"type":"array","items":{"type":"string"}}`, `{"additionalProperties":{"type":"array","items":{"type":"string"}}}`} {
		suffix := 1
		if strings.Contains(terminal, "additionalProperties") {
			suffix = 2
		}
		for _, cached := range []bool{false, true} {
			limit := 46 - suffix
			if cached {
				limit--
			}
			for _, count := range []int{limit, limit + 1} {
				props := []string{}
				for i := 0; i < count; i++ {
					target := "Value"
					if i > 0 {
						target = fmt.Sprintf("Hops/properties/h%d", i-1)
					}
					props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
				}
				body := `"Value":` + terminal + `,"Hops":{"properties":{` + strings.Join(props, ",") + `}}`
				use := fmt.Sprintf("Hops/properties/h%d", count-1)
				if cached {
					body += `,"Bridge":{"$ref":"#/components/schemas/` + use + `","nullable":false}`
					use = "Bridge"
				}
				body += `,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/` + use + `","nullable":false}},"required":["value"]}`
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+body+`}`), "Dto"))
				if count == limit {
					if err != nil {
						t.Fatal("namedcollectionsource48", suffix, cached, count, err)
					}
				} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
					t.Fatal("namedcollection source49 hidden", suffix, cached, count, err)
				}
			}
		}
	}
	for _, checked := range []bool{false, true} {
		for _, n := range []int{16, 17} {
			leaf := `{"type":"string"}`
			if checked {
				leaf = `{"type":"string","minLength":1}`
			}
			for i := 0; i < n; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			input := schemas(`{"AValues":` + leaf + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/AValues"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}}}`)
			r := request(input, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if !checked || n == 16 {
				if err != nil {
					t.Fatal("checkednativecollection16", checked, n, err)
				}
			} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0 {
				t.Fatal("newchain lostchecked17", checked, n, err)
			}
		}
	}
	// The actual root alias budget counts final named edge exactly once.
	for _, n := range []int{46, 47} {
		props := []string{}
		for i := 0; i < n; i++ {
			target := "Value"
			if i > 0 {
				target = fmt.Sprintf("Hops/properties/h%d", i-1)
			}
			props = append(props, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		doc := schemas(`{"Value":{"type":"array","items":{"type":"string"}},"Hops":{"properties":{` + strings.Join(props, ",") + `}},"Root":{"$ref":"#/components/schemas/Hops/properties/h` + fmt.Sprint(n-1) + `"}}`)
		got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
		if n == 46 {
			if err != nil {
				t.Fatal("root46hop+incoming1+array1=48", err)
			}
		} else if err == nil || len(got.Artifacts.Files) != 0 {
			t.Fatal("root49 accepted", err)
		}
	}
}

func TestNamedCollectionReferenceChainsForwardSourceAndNoDrop(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, terminal := range []string{`{"type":"array","items":{"$ref":"#/components/schemas/ZOther"}}`, `{"additionalProperties":{"$ref":"#/components/schemas/ZOther"}}`} {
			doc := schemas(`{"AValues":` + terminal + `,"ZOther":{"type":"array","items":{"type":"string"}},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/AValues"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Bridge"}}}}`)
			r := request(doc, "Dto")
			r.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("forwardoriginalnamedcollectionbacking", checked, terminal, err)
			}
		}
		for _, terminal := range []string{`{"type":"array","items":{"type":"string"},"minItems":-1}`, `{"additionalProperties":{"type":"string"},"required":["x"]}`, `{"type":"array","items":{"type":"string","default":"x"}}`} {
			doc := schemas(`{"Value":` + terminal + `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Value"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"}}`)
			r := request(doc, "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || len(got.Artifacts.Files) != 0 {
				t.Fatal("namedcollectionsourcegrammar dropped", checked, terminal, err)
			}
		}
	}
}
