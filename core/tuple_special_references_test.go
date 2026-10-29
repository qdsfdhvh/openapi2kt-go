package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

var tupleSpecialCases = []struct{ name, source, want, value string }{
	{"bigint", `{"x-native-type":"bigint"}`, "kotlin.Long", `[1,-9223372036854775808,9223372036854775807]`},
	{"byte", `{"type":"string","format":"byte"}`, "kotlin.ByteArray", `[[-128,0,127],[]]`},
	{"binary", `{"type":"string","format":"binary"}`, "kotlin.ByteArray", `[[1],[]]`},
	{"valueclass", `{"type":"string","x-kotlin-value-class":true}`, "ValueDto", `["ready"]`},
	{"native", `{"x-native-type":"org.example.scalarvalues.Money"}`, "org.example.scalarvalues.Money", `[]`},
	{"date-native", `{"x-native-type":"date"}`, "kotlin.time.Instant", `["2026-01-02T03:04:05Z"]`},
	{"raw", `{"type":"object","properties":{}}`, "kotlinx.serialization.json.JsonElement", `[null,{"description":{"$ref":"data"}},[1]]`},
	{"open", `{"type":"object","additionalProperties":true}`, "kotlinx.serialization.json.JsonObject", `[{"description":{"$ref":"data"}},{}]`},
	{"nullable-bigint", `{"x-native-type":"bigint","nullable":true}`, "kotlin.Long", `[null,7]`},
	{"nullable-bytes", `{"type":["string","null"],"format":"byte"}`, "kotlin.ByteArray", `[null,[]]`},
	{"nullable-valueclass", `{"type":["integer","null"],"format":"int64","x-kotlin-value-class":true}`, "ValueDto", `[null,7]`},
}

func TestTupleSpecialReferences(t *testing.T) {
	for _, tc := range tupleSpecialCases {
		for _, target := range []string{"Value", "Maybe", "Bridge"} {
			for _, tail := range []string{"", `,"items":true`, `,"items":false`, `,"items":{"type":"integer"}`} {
				input := schemas(`{"Value":` + tc.source + `,"Maybe":{"$ref":"#/components/schemas/Value","nullable":true},"Hops":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Bridge":{"$ref":"#/components/schemas/Hops/properties/value"},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]` + tail + `}}`)
				for _, checked := range []bool{false, true} {
					rq := request(input, "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(tc.name, target, tail, checked, err)
					}
					want := tc.want
					if tc.name == "open" {
						if !strings.Contains(source(t, got, "ValueDto"), "kotlinx.serialization.json.JsonObject") {
							t.Fatal("open dictionary lost its actual root backing")
						}
						if strings.Contains(source(t, got, "RowsDto"), "ValueDto") {
							want = "ValueDto"
						}
					}
					if tail == `,"items":true` {
						want = "kotlinx.serialization.json.JsonElement"
					} else if tail == `,"items":{"type":"integer"}` {
						want = "kotlin.Int"
					}
					if !strings.Contains(source(t, got, "RowsDto"), want) {
						t.Fatal(tc.name, target, tail, source(t, got, "RowsDto"))
					}
					if repeated, failure := core.GenerateWithDiagnostics(rq); failure != nil || !reflect.DeepEqual(got, repeated) {
						t.Fatal("special tuple nondeterminism", failure)
					}
				}
			}
		}
	}
}

func TestTupleSpecialOwnDefaults(t *testing.T) {
	for _, tc := range tupleSpecialCases {
		for _, checked := range []bool{false, true} {
			input := schemas(`{"Value":` + tc.source + `,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":` + tc.value + `},"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":` + tc.value + `},"required":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":` + tc.value + `}},"required":["required"]}}`)
			rq := request(input, "Dto")
			rq.Options.Validation = checked
			if _, err := core.GenerateWithDiagnostics(rq); err != nil {
				t.Fatal(tc.name, checked, err)
			}
		}
	}
	for _, tc := range []struct{ source, value string }{
		{tupleSpecialCases[0].source, `["1"]`},
		{tupleSpecialCases[1].source, `[[128]]`},
		{tupleSpecialCases[1].source, `["base64"]`},
		{tupleSpecialCases[3].source, `[1]`},
		{tupleSpecialCases[4].source, `["unproved-provider-value"]`},
		{tupleSpecialCases[5].source, `["invalid-timestamp"]`},
		{tupleSpecialCases[7].source, `[1]`},
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+tc.source+`,"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":`+tc.value+`}},"required":["rows"]}}`), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("required suppression hid invalid mapped annotation", tc.source, tc.value, err)
		}
	}
	for _, tc := range tupleSpecialCases[8:] {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+tc.source+`,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value","nullable":false}],"default":[null]}}`), "Dto"))
		if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("false consumer intent lost intrinsic root narrowing", tc.name, err)
		}
	}
}

func TestTupleSpecialErasedReferenceDepth(t *testing.T) {
	for _, tc := range tupleSpecialCases {
		for count := 42; count <= 48; count++ {
			parts := []string{`"Value":` + tc.source}
			for i := 0; i < count; i++ {
				target := "Value"
				if i > 0 {
					target = fmt.Sprintf("A%02d", i-1)
				}
				parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
			}
			ref := fmt.Sprintf(`{"$ref":"#/components/schemas/A%02d"}`, count-1)
			var ordinary error
			for _, role := range []string{"ordinary", "uniform", "raw", "scalar-tail"} {
				array := `"type":"array","items":` + ref
				if role != "ordinary" {
					array = `"type":"array","prefixItems":[` + ref + `]`
					if role == "raw" {
						array += `,"items":true`
					} else if role == "scalar-tail" {
						array += `,"items":{"type":"integer"}`
					}
				}
				got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`,"Rows":{`+array+`},"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`), "Dto"))
				if role == "ordinary" {
					ordinary = err
				} else if (ordinary == nil) != (err == nil) {
					t.Fatal("erased special source depth differs", tc.name, count, role, ordinary, err)
				}
				if err != nil && (err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
					t.Fatal(tc.name, count, role, err)
				}
			}
		}
	}
}

func TestTupleSpecialSharedSourceBudget(t *testing.T) {
	for _, count := range []int{1, 1100} {
		for _, positional := range []bool{false, true} {
			parts := []string{}
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("Value%04d", i)
				parts = append(parts, `"`+name+`":{"x-native-type":"bigint"}`)
				array := `"type":"array","items":{"$ref":"#/components/schemas/` + name + `"}`
				if positional {
					array = `"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + name + `"}],"items":true`
				}
				parts = append(parts, fmt.Sprintf(`"Rows%04d":{%s}`, i, array))
			}
			got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
			if count == 1100 && positional {
				if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
					t.Fatal("distinct mapped references escaped shared tuple budget", err)
				}
			} else if err != nil {
				t.Fatal("original ordinary/small tuple source control rejected", count, positional, err)
			}
		}
	}
}

func TestTupleSpecialValueClassDefaultChecks(t *testing.T) {
	for _, value := range []string{`["ready"]`, `["x"]`} {
		rq := request(schemas(`{"Value":{"type":"string","x-kotlin-value-class":true,"minLength":3},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":`+value+`}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (value == `["ready"]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("wrapper-own constructor checks lost in tuple annotation", value, err)
		}
	}
}

func TestTupleOpenDictionaryDefaultCounts(t *testing.T) {
	for _, value := range []string{`[{}]`, `[{"present":true}]`} {
		rq := request(schemas(`{"Value":{"type":"object","additionalProperties":true,"minProperties":1},"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Value"}],"default":`+value+`}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if (err == nil) != (value != `[{}]`) || err != nil && !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("retained JsonObject counts lost in outer tuple annotation", value, err)
		}
	}
}

func TestTupleCapturedNamedAliasBudget(t *testing.T) {
	for _, positional := range []bool{false, true} {
		parts := []string{`"Value":{"type":"integer","enum":[1,2]}`}
		for i := 0; i < 1400; i++ {
			alias := fmt.Sprintf("A%04d", i)
			parts = append(parts, `"`+alias+`":{"$ref":"#/components/schemas/Value"}`)
			array := `"type":"array","items":{"$ref":"#/components/schemas/` + alias + `"}`
			if positional {
				array = `"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + alias + `"}]`
			}
			parts = append(parts, fmt.Sprintf(`"Rows%04d":{%s}`, i, array))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if positional {
			if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("actual named-alias captures escaped existing shared cap", err)
			}
		} else if err != nil {
			t.Fatal("ordinary list original-source control rejected", err)
		}
	}
}
