package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestFalseDTOOwnerRootOwnDefaults(t *testing.T) {
	for _, input := range []string{
		`{"Target":{"properties":{"rows":{"type":"array","items":false}}},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"rows":[]}}}}`,
		`{"Target":{"properties":{"blocked":false}},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{}}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil {
			t.Errorf("root annotation could not borrow actual false DTO target: %v", err)
			continue
		}
		if !strings.Contains(source(t, report, "OuterDto"), "public fun default()") {
			t.Fatal("missing actual owner root factory")
		}
		if strings.Contains(source(t, report, "TargetDto"), "public fun default()") {
			t.Fatal("borrowed root default was inherited by target")
		}
	}
}

func TestFalseDTOOwnerRootProfilesAndFactoryOrder(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/false-dto-owner-root-defaults.json")
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), input...)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for bits := 0; bits < 8; bits++ {
				rq := request(input, "Dto")
				rq.Options.ModelStyle = style
				rq.Options.Validation = checked
				rq.Options.DeduplicateModels = bits&1 != 0
				rq.Options.HoistSharedShapes = bits&2 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				rq.Options.MergeIdenticalModels = bits&4 != 0
				report, err := core.GenerateWithDiagnostics(rq)
				if err != nil {
					t.Fatal(style, checked, bits, err)
				}
				for _, name := range []string{"OuterDto", "LaterOwnerDto", "NullableMemberDto", "NullableOwnerDto", "OwnerWithoutFactoryDto", "CountedOwnerDto"} {
					if !strings.Contains(source(t, report, name), "public fun default()") {
						t.Fatal("missing actual root factory", name)
					}
				}
				if strings.Contains(source(t, report, "WithoutFactoryDto"), "public fun default()") {
					t.Fatal("owner annotation inherited by target")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(report, again) || !bytes.Equal(input, before) {
					t.Fatal("root-edge planning nondeterministic or input mutated", e)
				}
			}
		}
	}
	for _, names := range [][2]string{{"AOwner", "ZTarget"}, {"ZOwner", "ATarget"}} {
		for _, valid := range []bool{false, true} {
			value := `[]`
			if !valid {
				value = `[null]`
			}
			input := schemas(fmt.Sprintf(`{"%s":{"properties":{"target":{"$ref":"#/components/schemas/%s"}},"default":{"target":{"rows":[]}}},"%s":{"properties":{"rows":{"type":"array","items":false}},"default":{"rows":%s}}}`, names[0], names[1], names[1], value))
			report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
			if valid {
				if err != nil {
					t.Fatal("factory admission depends on alphabetical order", names, err)
				}
			} else {
				if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
					t.Fatal("borrow bypassed independent invalid target factory", names, err)
				}
				d := err.(core.Diagnostics)[0]
				if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/"+names[1]+"/default/rows/0" {
					t.Fatal("independent target source Pointer lost", d)
				}
			}
		}
	}
}

func TestFalseDTOOwnerRootValuesAndClosedRoles(t *testing.T) {
	for _, tc := range []struct{ properties, required, value, bounds, pointer string }{
		{`{"rows":{"type":"array","items":false}}`, `[]`, `{"rows":[null]}`, ``, "/default/target/rows/0"},
		{`{"rows":{"type":"array","items":false}}`, `[]`, `{"rows":null}`, ``, "/default/target/rows"},
		{`{"rows":{"type":"array","items":false,"default":[]}}`, `["rows"]`, `{}`, ``, "/default/target"},
		{`{"blocked":false}`, `[]`, `{"blocked":null}`, ``, "/default/target/blocked"},
		{`{"rows":{"type":"array","items":false}}`, `[]`, `null`, ``, "/default/target"},
		{`{"rows":{"type":"array","items":false}}`, `[]`, `{}`, `,"minProperties":1`, "/default/target"},
		{`{"rows":{"type":"array","minItems":1,"items":false}}`, `[]`, `{"rows":[]}`, ``, "/default/target/rows"},
	} {
		input := schemas(fmt.Sprintf(`{"Target":{"required":%s,"properties":%s%s},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":%s}}}`, tc.required, tc.properties, tc.bounds, tc.value))
		rq := request(input, "Dto")
		rq.Options.Validation = true
		report, err := core.GenerateWithDiagnostics(rq)
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("invalid owner literal admitted", tc, err)
		}
		d := err.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Outer"+tc.pointer {
			t.Fatal("owner literal Pointer lost", tc, d)
		}
	}
	for _, role := range []string{
		`"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Alias"}},"default":{"target":{}}}`,
		`"Outer":{"properties":{"nested":{"properties":{"target":{"$ref":"#/components/schemas/Target"}}}},"default":{"nested":{"target":{}}}}`,
		`"Outer":{"allOf":[{"properties":{"target":{"$ref":"#/components/schemas/Target"}}}],"default":{"target":{}}}`,
		`"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{}}},"default":{}}`,
		`"Outer":{"properties":{"targets":{"type":"array","items":{"$ref":"#/components/schemas/Target"}}},"default":{"targets":[{}]}}`,
	} {
		input := schemas(`{"Target":{"properties":{"blocked":false}},` + role + `}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if role == `"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Alias"}},"default":{"target":{}}}` {
			if err != nil || !strings.Contains(source(t, report, "OuterDto"), "public fun default()") {
				t.Fatal("original94 purealias root-member graduation", err)
			}
			continue
		}
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("direct root-member scope widened", role, err)
		}
	}
}

func TestFalseDTOOwnerRootBudgets(t *testing.T) {
	for _, depth := range []int{46, 47} {
		var leaf any = false
		for i := 0; i < depth; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		target := map[string]any{"properties": map[string]any{"rows": leaf}}
		owner := map[string]any{"properties": map[string]any{"target": map[string]any{"$ref": "#/components/schemas/Target"}}, "default": map[string]any{}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Outer": owner}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if depth == 46 {
			if err != nil {
				t.Fatal("bounded original root-member source depth", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("source48 reset at borrowed root member", err)
		}
	}
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"$ref": "#/components/schemas/Target"}
		}
		owners := map[string]any{"Target": map[string]any{"properties": map[string]any{"blocked": false}}}
		for i := 0; i < count; i++ {
			owners[fmt.Sprintf("O%02d", i)] = map[string]any{"properties": props, "default": map[string]any{}}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": owners}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("bounded root source capture", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("cumulative false-work budget reset", err)
		}
	}
	for _, count := range []int{6549, 6550} {
		target := map[string]any{"properties": map[string]any{"labels": map[string]any{"type": "array", "items": map[string]any{"type": "array", "nullable": true, "items": false}}}}
		owner := map[string]any{"properties": map[string]any{"target": map[string]any{"$ref": "#/components/schemas/Target"}}, "default": map[string]any{"target": map[string]any{"labels": make([]any, count)}}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Outer": owner}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6549 {
			if err != nil {
				t.Fatal("bounded32768byte root literal", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("32KiB root value budget reset", err)
		}
	}
}
