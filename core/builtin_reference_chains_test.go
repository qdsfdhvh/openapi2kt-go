package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func builtinReferenceChainsInput(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/http/builtin-reference-chains.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBuiltinReferenceChainsCanonicalViews(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			r := request(builtinReferenceChainsInput(t), "Dto")
			r.Options.ModelStyle, r.Options.Validation = style, checked
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct{ alias, scalar string }{
				{"Text", "String"}, {"Flag", "Boolean"}, {"Count", "Int"}, {"Wide", "Long"}, {"Ratio", "Float"}, {"Measure", "Double"},
			} {
				alias := tc.alias + "BridgeDto"
				if !strings.Contains(source(t, got, alias), "public typealias "+alias+" = kotlin."+tc.scalar+"?\n") {
					t.Fatal(alias, source(t, got, alias))
				}
			}
			if !strings.Contains(source(t, got, "TextBridgeFalseDto"), "= kotlin.String\n") || !strings.Contains(source(t, got, "AgainBridgeDto"), "= kotlin.String?\n") {
				t.Fatal("root or entering internal nullable override lost")
			}
			patch := source(t, got, "PatchBuiltinInputDto")
			if !strings.Contains(patch, "Patch<kotlin.String>") || !strings.Contains(patch, "Patch<kotlin.Long>") || !strings.Contains(patch, "count: kotlin.Int?,") || !strings.Contains(patch, "narrow: kotlin.String,") {
				t.Fatal(patch)
			}
			api := source(t, got, "DefaultApi")
			if strings.Count(api, "public suspend fun echoBuiltin(") != 2 || strings.Count(api, "public suspend fun echoBuiltin2(") != 1 || strings.Contains(api, "rawValue") {
				t.Fatal(api)
			}
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("nondeterministic builtin chain", err)
			}
		}
	}
}

func TestBuiltinReferenceChainsDirectAndLegacyTerminals(t *testing.T) {
	for _, terminal := range []string{`{"type":"string"}`, `{"type":"string","format":"uuid"}`, `{"type":"boolean"}`, `{"type":"integer"}`, `{"type":"integer","format":"int64"}`, `{"type":"number","format":"float"}`, `{"type":"number"}`} {
		for _, alias := range []bool{false, true} {
			for _, root := range []bool{false, true} {
				body := `"Value":` + terminal
				target := "Value"
				if alias {
					body += `,"Alias":{"$ref":"#/components/schemas/Value"}`
					target = "Alias"
				}
				body += `,"Hops":{"properties":{"value":{"$ref":"#/components/schemas/` + target + `"}}}`
				if root {
					body += `,"Bridge":{"$ref":"#/components/schemas/Hops/properties/value","nullable":false}`
				} else {
					body += `,"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Hops/properties/value"}},"required":["value"]}`
				}
				for _, checked := range []bool{false, true} {
					r := request(schemas(`{`+body+`}`), "Dto")
					r.Options.Validation = checked
					_, err := core.GenerateWithDiagnostics(r)
					if alias && !checked {
						if err == nil || !strings.Contains(err.Error(), "E_SCHEMA_UNSUPPORTED") {
							t.Fatal("legacy alias policy broadened", err)
						}
					} else if err != nil {
						t.Fatal(terminal, alias, root, checked, err)
					}
				}
			}
		}
	}
}
