package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestFalseDTOOwnerAliasRootDefaults(t *testing.T) {
	for _, input := range []string{
		`{"Target":{"properties":{"rows":{"type":"array","items":false}}},"Alias":{"$ref":"#/components/schemas/Target"},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Alias"}},"default":{"target":{"rows":[]}}}}`,
		`{"Target":{"properties":{"blocked":false}},"Alias":{"$ref":"#/components/schemas/Target","nullable":true},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Alias"}},"default":{"target":null}}}`,
	} {
		report, err := core.GenerateWithDiagnostics(request(schemas(input), "Dto"))
		if err != nil {
			t.Errorf("original alias root-member path rejected: %v", err)
			continue
		}
		if !strings.Contains(source(t, report, "OuterDto"), "public fun default()") {
			t.Fatal("missing original owner factory")
		}
	}
}

func TestFalseDTOOwnerAliasDepthAndClosedPaths(t *testing.T) {
	for _, count := range []int{46, 47} {
		defs := map[string]any{"Target": map[string]any{"properties": map[string]any{"blocked": false}}}
		last := "Target"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			defs[name] = map[string]any{"$ref": "#/components/schemas/" + last}
			last = name
		}
		defs["Outer"] = map[string]any{"properties": map[string]any{"target": map[string]any{"$ref": "#/components/schemas/" + last}}, "default": map[string]any{"target": map[string]any{}}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": defs}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 46 {
			if err != nil {
				t.Fatal("bounded original alias path", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("alias source48 reset", err)
		}
	}
	for _, alias := range []string{
		`"Alias":{"$ref":"#/components/schemas/Target","default":{}}`,
		`"Alias":{"$ref":"#/components/schemas/Alias"}`,
		`"Alias":{"$ref":"#/components/schemas/Target/properties/child"}`,
	} {
		target := `"Target":{"properties":{"blocked":false}}`
		if strings.Contains(alias, "/properties/child") {
			target = `"Target":{"properties":{"blocked":false,"child":{"properties":{"blocked":false}}}}`
		}
		input := schemas(`{` + target + `,` + alias + `,"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Alias"}},"default":{"target":{}}}}`)
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if err == nil || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("non-pure alias path admitted", alias, err)
		}
	}
}

func TestFalseDTOOwnerAliasSharedCaptureAndValueBudgets(t *testing.T) {
	for _, count := range []int{8, 20} {
		props := map[string]any{}
		for i := 0; i < 240; i++ {
			props[fmt.Sprintf("p%03d", i)] = map[string]any{"$ref": "#/components/schemas/Alias"}
		}
		defs := map[string]any{"Target": map[string]any{"properties": map[string]any{"blocked": false}}, "Alias": map[string]any{"$ref": "#/components/schemas/Target"}}
		for i := 0; i < count; i++ {
			defs[fmt.Sprintf("O%02d", i)] = map[string]any{"properties": props, "default": map[string]any{}}
		}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": defs}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 8 {
			if err != nil {
				t.Fatal("repeated original alias capture not shared", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("shared capture cap reset", err)
		}
	}
	for _, count := range []int{6549, 6550} {
		target := map[string]any{"properties": map[string]any{"labels": map[string]any{"type": "array", "items": map[string]any{"type": "array", "nullable": true, "items": false}}}}
		owner := map[string]any{"properties": map[string]any{"target": map[string]any{"$ref": "#/components/schemas/Alias"}}, "default": map[string]any{"target": map[string]any{"labels": make([]any, count)}}}
		input, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "components": map[string]any{"schemas": map[string]any{"Target": target, "Alias": map[string]any{"$ref": "#/components/schemas/Target"}, "Outer": owner}}})
		report, err := core.GenerateWithDiagnostics(request(input, "Dto"))
		if count == 6549 {
			if err != nil {
				t.Fatal("exact32768 alias owner literal", err)
			}
		} else if err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(report, core.GenerationReport{}) {
			t.Fatal("alias owner32KiB cap reset", err)
		}
	}
}

func TestFalseDTOOwnerAliasNullableAndOptions(t *testing.T) {
	for _, tc := range []struct {
		terminal, near, far, field, value string
		valid                             bool
	}{
		{"false", "", "", "", `{}`, true},
		{"false", "", `,"nullable":true`, "", `null`, true},
		{"false", `,"nullable":false`, `,"nullable":true`, "", `null`, false},
		{"false", `,"nullable":false`, `,"nullable":true`, `,"nullable":true`, `null`, true},
		{"false", `,"nullable":true`, "", `,"nullable":false`, `null`, false},
		{"true", "", "", "", `null`, true},
		{"true", `,"nullable":false`, "", "", `null`, false},
	} {
		input := schemas(fmt.Sprintf(`{"Target":{"type":"object","nullable":%s,"properties":{"blocked":false}},"Far":{"$ref":"#/components/schemas/Target"%s},"Near":{"$ref":"#/components/schemas/Far"%s},"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Near"%s}},"default":{"target":%s}}}`, tc.terminal, tc.far, tc.near, tc.field, tc.value))
		for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
			for _, checked := range []bool{false, true} {
				for mask := 0; mask < 8; mask++ {
					rq := request(input, "Dto")
					rq.Options.ModelStyle = style
					rq.Options.Validation = checked
					rq.Options.DeduplicateModels = mask&1 != 0
					rq.Options.HoistSharedShapes = mask&2 != 0
					rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
					rq.Options.MergeIdenticalModels = mask&4 != 0
					report, err := core.GenerateWithDiagnostics(rq)
					if (err == nil) != tc.valid {
						t.Fatal(tc, style, checked, mask, err)
					}
					if err != nil {
						if !reflect.DeepEqual(report, core.GenerationReport{}) {
							t.Fatal("partial alias artifacts", err)
						}
						d := err.(core.Diagnostics)[0]
						if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Outer/default/target" {
							t.Fatal("nearest nullable Pointer lost", tc, d)
						}
					} else {
						again, e := core.GenerateWithDiagnostics(rq)
						if e != nil || !reflect.DeepEqual(report, again) {
							t.Fatal("alias factory nondeterministic", e)
						}
					}
				}
			}
		}
	}
}
