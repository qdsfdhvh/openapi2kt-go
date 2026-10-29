package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestTupleBooleanReferences(t *testing.T) {
	for _, tc := range []struct{ prefix, tail, want string }{
		{`[{"$ref":"#/components/schemas/Empty"}]`, "", "EmptyDto>"},
		{`[{"$ref":"#/components/schemas/Empty"},{"$ref":"#/components/schemas/Alias"}]`, ",\"items\":false", "EmptyDto>"},
		{`[{"$ref":"#/components/schemas/Maybe"}]`, "", "EmptyDto?>"},
		{`[{"$ref":"#/components/schemas/Maybe","nullable":false}]`, "", "EmptyDto>"},
		{`[{"$ref":"#/components/schemas/Empty"},{"$ref":"#/components/schemas/Other"}]`, "", "JsonElement>"},
		{`[{"$ref":"#/components/schemas/Empty"}]`, `,"items":true`, "JsonElement>"},
		{`[{"$ref":"#/components/schemas/Any"}]`, "", "JsonElement>"},
		{`[{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Empty"}`, "EmptyDto>"},
		{`[{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Maybe"}`, "EmptyDto?>"},
		{`[{"type":"integer"}]`, `,"items":{"$ref":"#/components/schemas/Any"}`, "JsonElement>"},
	} {
		for _, checked := range []bool{false, true} {
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				rq := request(schemas(`{"Empty":false,"Other":false,"Alias":{"$ref":"#/components/schemas/Empty"},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true},"Any":true,"Rows":{"type":"array","prefixItems":`+tc.prefix+tc.tail+`}}`), "Dto")
				rq.Options.Validation, rq.Options.ModelStyle = checked, style
				got, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(tc, checked, style, err)
				}
				if !strings.Contains(source(t, got, "RowsDto"), tc.want) {
					t.Fatal(tc, source(t, got, "RowsDto"))
				}
				again, repeated := core.GenerateWithDiagnostics(rq)
				if repeated != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("Boolean tuple references nondeterministic", repeated)
				}
			}
		}
	}
}

func TestTupleBooleanReferenceOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		leaf, value string
		valid       bool
	}{
		{"Empty", `[]`, true}, {"Empty", `[null]`, false}, {"Empty", `[1]`, false},
		{"Maybe", `[null,null]`, true}, {"Maybe", `[{}]`, false},
		{"Any", `[null,{"default":"data"},1]`, true},
	} {
		tuple := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + tc.leaf + `"}],"default":` + tc.value + `}`
		for _, root := range []bool{false, true} {
			owner := `"Rows":` + tuple
			if !root {
				owner = `"Holder":{"properties":{"rows":` + tuple + `},"required":["rows"]}`
			}
			input := schemas(`{"Empty":false,"Maybe":{"$ref":"#/components/schemas/Empty","nullable":true},"Any":true,` + owner + `}`)
			_, err := core.Generate(request(input, "Dto"))
			if (err == nil) != tc.valid {
				if err != nil {
					t.Fatalf("Boolean reference default %v root=%v: %v at %s", tc, root, err, *err.(core.Diagnostics)[0].Pointer)
				}
				t.Fatal("invalid Boolean reference default was accepted", tc, root)
			}
			if err != nil && err.(core.Diagnostics)[0].Code != "E_SCHEMA_INVALID" {
				t.Fatal("default did not reach actual Boolean leaf", tc, err)
			}
		}
	}
}

func TestTupleBooleanReferenceDiscardedSources(t *testing.T) {
	for _, extra := range []string{`,"items":true`, `,"items":{"type":"string"}`} {
		failure(t, schemas(`{"Empty":false,"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Empty","unknownTupleSource":true}]`+extra+`}}`), "E_SCHEMA_UNSUPPORTED", "/components/schemas/Rows/prefixItems/0/unknownTupleSource")
	}
}

func TestTupleBooleanReferenceDepthAndCapture(t *testing.T) {
	for _, literal := range []string{"false", "true"} {
		for _, count := range []int{47, 48} {
			leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Leaf"}]}`
			for i := 1; i < count; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			_, err := core.Generate(request(schemas(`{"Leaf":`+literal+`,"Rows":`+leaf+`}`), "Dto"))
			if count == 47 && err != nil || count == 48 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
				t.Fatal("Boolean reference original source depth", literal, count, err)
			}
		}
		for _, hops := range []int{0, 2} {
			for _, count := range []int{45 - hops, 46 - hops} {
				for _, tail := range []string{"", `,"items":true`, `,"items":{"type":"integer"}`} {
					parts := []string{`"Leaf":` + literal}
					target := "Leaf"
					for i := 0; i < hops; i++ {
						parts = append(parts, fmt.Sprintf(`"Alias%d":{"$ref":"#/components/schemas/%s"}`, i, target))
						target = fmt.Sprintf("Alias%d", i)
					}
					leaf := `{"type":"array","prefixItems":[{"$ref":"#/components/schemas/` + target + `"}]` + tail + `}`
					for i := 1; i < count; i++ {
						leaf = `{"type":"array","items":` + leaf + `}`
					}
					parts = append(parts, `"Rows":`+leaf, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`)
					for _, forward := range []bool{false, true} {
						if forward {
							parts[0], parts[len(parts)-1] = parts[len(parts)-1], parts[0]
						}
						_, err := core.Generate(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
						if count == 45-hops && err != nil || count == 46-hops && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH") {
							t.Fatal("Boolean reference cached incoming depth", literal, hops, count, tail, forward, err)
						}
					}
				}
			}
		}
	}
	parts := []string{`"Empty":false`}
	for i := 0; i < 2000; i++ {
		parts = append(parts, fmt.Sprintf(`"R%04d":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Empty"}],"items":true}`, i))
	}
	got, err := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
	if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("cumulative discarded Boolean reference capture reset", err)
	}
}
