package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedScalarFieldDefaultsReferenceFailuresAndCacheHeight(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, tc := range []struct{ ref, code string }{
			{"#/components/schemas/Missing", "E_REF_MISSING"},
			{"https://example.test/schema.json#/Count", "E_REF_UNSUPPORTED"},
		} {
			r := request(schemas(`{"Holder":{"properties":{"v":{"$ref":"`+tc.ref+`","default":null}}}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, err)
			}
			// External schema references keep the original source diagnostic family.
			d := err.(core.Diagnostics)[0]
			if *d.Pointer != "/components/schemas/Holder/properties/v" {
				t.Fatal(tc, d)
			}
			if tc.ref[0] == '#' && d.Code != tc.code {
				t.Fatal(tc, d)
			}
		}
		r := request(schemas(`{"Hops":{"properties":{"a":{"$ref":"#/components/schemas/Hops/properties/b"},"b":{"$ref":"#/components/schemas/Hops/properties/a"}}},"Holder":{"properties":{"v":{"$ref":"#/components/schemas/Hops/properties/a","default":null}}}}`), "Dto")
		r.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(r)
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("transparent cycle", err)
		}
	}
	for _, count := range []int{46, 47} {
		hops := []string{}
		for i := 0; i < count; i++ {
			target := "Source"
			if i > 0 {
				target = fmt.Sprintf("Hops/properties/h%d", i-1)
			}
			hops = append(hops, fmt.Sprintf(`"h%d":{"$ref":"#/components/schemas/%s","nullable":false}`, i, target))
		}
		for _, own := range []bool{false, true} {
			annotation := ""
			if own {
				annotation = `,"default":"source"`
			}
			doc := schemas(fmt.Sprintf(`{"Source":{"type":"string","nullable":true},"Hops":{"properties":{%s}},"Holder":{"properties":{"first":{"$ref":"#/components/schemas/Hops/properties/h%d"%s},"cached":{"$ref":"#/components/schemas/Hops/properties/h%d"%s}}}}`, strings.Join(hops, ","), count-1, annotation, count-1, annotation))
			got, err := core.GenerateWithDiagnostics(request(doc, "Dto"))
			if count == 46 && err != nil || count == 47 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || len(got.Artifacts.Files) != 0) {
				t.Fatal(count, own, err)
			}
		}
	}
}
