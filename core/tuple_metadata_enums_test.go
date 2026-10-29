package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

var tupleStringMetadata = []string{
	`"format":""`, `"format":"date"`, `"format":"date-time"`, `"format":"byte"`, `"format":"binary"`, `"format":"uuid"`,
	`"x-native-type":"date"`, `"x-native-type":"bigint"`, `"x-native-type":"org.example.provider.Amount"`,
}

func TestTupleMetadataEnumSourcesAndDefaults(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready","future"],` + metadata + `}`
		for _, selected := range []string{leaf, `{"$ref":"#/components/schemas/Value"}`, `{"$ref":"#/components/schemas/Maybe","nullable":false}`, `{"$ref":"#/components/schemas/Value","nullable":true}`} {
			for _, tail := range []string{"", `,"items":false`, `,"items":true`, `,"items":{"type":"integer"}`, `,"items":` + selected} {
				value := `["ready","future"]`
				if tail == `,"items":{"type":"integer"}` {
					value = `[7]`
				}
				if tail == `,"items":true` {
					value = `[{"enum":"data"},null]`
				}
				tuple := `{"type":"array","prefixItems":[` + selected + `]` + tail + `,"default":` + value + `}`
				input := schemas(`{"Value":` + leaf + `,"Maybe":{"$ref":"#/components/schemas/Value"},"Rows":` + tuple + `,"Alias":{"$ref":"#/components/schemas/Rows","default":[]},"Holder":{"properties":{"rows":` + tuple + `}}}`)
				for _, checked := range []bool{false, true} {
					rq := request(input, "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err != nil {
						t.Fatal(metadata, selected, tail, checked, err, *err.(core.Diagnostics)[0].Pointer)
					}
					if !strings.Contains(source(t, got, "ValueDto"), `sealed class ValueDto`) {
						t.Fatal("metadata erased lexical enum", metadata)
					}
					if again, err := core.GenerateWithDiagnostics(rq); err != nil || !reflect.DeepEqual(got, again) {
						t.Fatal("nondeterministic metadata enum", err)
					}
				}
			}
		}
	}
}

func TestTupleMetadataEnumDefaultMembership(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready"],` + metadata + `}`
		for _, selected := range []string{leaf, `{"$ref":"#/components/schemas/Value"}`} {
			for _, value := range []string{`["other"]`, `[1]`, `[null]`} {
				for _, checked := range []bool{false, true} {
					rq := request(schemas(`{"Value":`+leaf+`,"Holder":{"properties":{"rows":{"type":"array","prefixItems":[`+selected+`],"default":`+value+`}},"required":["rows"]}}`), "Dto")
					rq.Options.Validation = checked
					got, err := core.GenerateWithDiagnostics(rq)
					if err == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal("invalid metadata enum default accepted", metadata, selected, value, err)
					}
					if d := err.(core.Diagnostics)[0]; d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder/properties/rows/default/0" {
						t.Fatal("lost original default diagnostic", metadata, err)
					}
				}
			}
		}
	}
}

func TestTupleMetadataEnumDiscardedSourceGuards(t *testing.T) {
	for _, tc := range []struct{ extra, pointer string }{
		{`"format":17`, "/format"}, {`"format":"email"`, "/format"},
		{`"x-native-type":"unsafe.Type()"`, "/x-native-type"},
		{`"format":"date","minLength":1`, "/minLength"},
		{`"format":"date","default":"ready"`, "/default"},
		{`"format":"date","x-kotlin-value-class":true`, "/x-kotlin-value-class"},
	} {
		for _, tail := range []string{`,"items":true`, `,"items":{"type":"boolean"}`} {
			input := schemas(`{"Rows":{"type":"array","prefixItems":[{"type":"string","enum":["ready"],` + tc.extra + `}]` + tail + `}}`)
			_, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if err == nil {
				t.Fatal("discarded source skipped metadata validation", tc)
			}
			if d := err.(core.Diagnostics)[0]; d.Pointer == nil || *d.Pointer != "/components/schemas/Rows/prefixItems/0"+tc.pointer {
				t.Fatal(tc, err)
			}
		}
	}
	for _, metadata := range tupleStringMetadata {
		// Metadata never grants the plain enum's inferred/explicit null vocabulary.
		_, err := core.Generate(request(schemas(`{"Rows":{"type":"array","prefixItems":[{"type":["string","null"],"enum":[null,"ready"],`+metadata+`}],"items":true}}`), "Dto"))
		if err == nil {
			t.Fatal("metadata gained null vocabulary admission", metadata)
		}
	}
}

func TestTupleMetadataEnumErasedDepth(t *testing.T) {
	for _, count := range []int{43, 44, 45, 46, 47, 48} {
		parts := []string{`"Value":{"type":"string","enum":["ready"],"format":"date"}`}
		for i := 0; i < count; i++ {
			target := "Value"
			if i > 0 {
				target = fmt.Sprintf("A%02d", i-1)
			}
			parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		ref := fmt.Sprintf(`{"$ref":"#/components/schemas/A%02d"}`, count-1)
		var ordinary error
		for _, role := range []string{"ordinary", "uniform", "raw", "tail"} {
			array := `"type":"array","items":` + ref
			if role != "ordinary" {
				array = `"type":"array","prefixItems":[` + ref + `]`
				if role == "raw" {
					array += `,"items":true`
				}
				if role == "tail" {
					array += `,"items":{"type":"integer"}`
				}
			}
			_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`,"Rows":{`+array+`}}`), "Dto"))
			if role == "ordinary" {
				ordinary = err
			} else if (ordinary == nil) != (err == nil) {
				t.Fatal("tuple source lost erased alias depth", count, role, ordinary, err)
			}
			if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" {
				t.Fatal(count, role, err)
			}
		}
	}
}

func TestTupleMetadataEnumNullableDefaults(t *testing.T) {
	for _, metadata := range tupleStringMetadata {
		leaf := `{"type":"string","enum":["ready"],` + metadata + `}`
		for _, selected := range []string{strings.TrimSuffix(leaf, "}") + `,"nullable":true}`, `{"$ref":"#/components/schemas/Value","nullable":true}`} {
			for _, checked := range []bool{false, true} {
				tuple := `{"type":"array","prefixItems":[` + selected + `],"default":[null,"ready"]}`
				rq := request(schemas(`{"Value":`+leaf+`,"Rows":`+tuple+`,"Holder":{"properties":{"rows":`+tuple+`}}}`), "Dto")
				rq.Options.Validation = checked
				if _, err := core.Generate(rq); err != nil {
					t.Fatal(metadata, selected, checked, err)
				}
			}
		}
	}
}

func TestTupleMetadataEnumProjectionAndBudgets(t *testing.T) {
	leaf := `{"type":"string","format":"date","enum":["ready"]}`
	for _, tc := range []struct{ prefix, want string }{
		{`[{"$ref":"#/components/schemas/Value"},{"$ref":"#/components/schemas/Alias"}]`, "ValueDto"},
		{`[` + leaf + `,` + leaf + `]`, "kotlinx.serialization.json.JsonElement"},
		{`[{"$ref":"#/components/schemas/Value"},{"$ref":"#/components/schemas/Other"}]`, "kotlinx.serialization.json.JsonElement"},
		{`[{"$ref":"#/components/schemas/Value"},{"$ref":"#/components/schemas/Value","nullable":true}]`, "kotlinx.serialization.json.JsonElement"},
	} {
		got, err := core.GenerateWithDiagnostics(request(schemas(`{"Value":`+leaf+`,"Other":`+leaf+`,"Alias":{"$ref":"#/components/schemas/Value"},"Rows":{"type":"array","prefixItems":`+tc.prefix+`}}`), "Dto"))
		if err != nil || !strings.Contains(source(t, got, "RowsDto"), tc.want) {
			t.Fatal(tc, err)
		}
	}
	values := make([]string, 240)
	for i := range values {
		values[i] = fmt.Sprintf(`"v%d"`, i)
	}
	for _, count := range []int{1, 64} {
		entries := make([]string, count)
		for i := range entries {
			entries[i] = fmt.Sprintf(`"Rows%d":{"type":"array","prefixItems":[{"type":"string","format":"date","enum":[%s]}],"items":true}`, i, strings.Join(values, ","))
		}
		got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto"))
		if count == 1 && err != nil || count == 64 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("metadata source budget", count, err)
		}
	}
}
