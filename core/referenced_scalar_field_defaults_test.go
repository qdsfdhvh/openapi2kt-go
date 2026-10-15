package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestReferencedScalarFieldDefaultsFixture(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/referenced-scalar-field-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, checked := range []bool{false, true} {
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			r := request(input, "Dto")
			r.Options.Validation = checked
			r.Options.ModelStyle = style
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			text := source(t, got, "HolderDto")
			if !strings.Contains(text, "public val required: kotlin.Long,") || !strings.Contains(text, "public val count: kotlin.Long? = 2L") || strings.Contains(text, "default tree") {
				t.Fatal(text)
			}
			patch := source(t, got, "PatchInputDto")
			if !strings.Contains(patch, "Patch.Absent") || !strings.Contains(patch, "public val ordinary: kotlin.Long? = 2L") {
				t.Fatal(patch)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal(err)
			}
		}
	}
}

func TestReferencedScalarFieldDefaultsBaseline(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, nullableRoot := range []bool{false, true} {
			root := `{"type":"integer","format":"int64"}`
			if nullableRoot {
				root = `{"type":"integer","format":"int64","nullable":true}`
			}
			for _, own := range []bool{false, true} {
				field := `{"$ref":"#/components/schemas/Count"}`
				if own {
					field = `{"$ref":"#/components/schemas/Count","default":2}`
				}
				r := request(schemas(`{"Count":`+root+`,"Holder":{"properties":{"count":`+field+`}}}`), "Dto")
				r.Options.Validation = checked
				got, err := core.GenerateWithDiagnostics(r)
				t.Logf("checked=%v nullableRoot=%v ownDefault=%v error=%v", checked, nullableRoot, own, err)
				if err != nil {
					t.Fatal(err)
				}
				want := "public val count: kotlin.Long? = null"
				if own {
					want = "public val count: kotlin.Long? = 2L"
				}
				if !strings.Contains(source(t, got, "HolderDto"), want) {
					t.Fatal(want, got)
				}
			}
		}
	}
}
