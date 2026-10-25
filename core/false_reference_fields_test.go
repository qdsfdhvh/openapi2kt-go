package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestFalseReferenceOptionalFieldMinimal(t *testing.T) {
	input := schemas(`{"Denied":false,"Holder":{"properties":{"blocked":{"$ref":"#/components/schemas/Denied"}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(checked, err)
		}
		model := source(t, got, "HolderDto")
		if !strings.Contains(model, "EncodeDefault(kotlinx.serialization.EncodeDefault.Mode.NEVER)") || !strings.Contains(model, "DeniedDto.AbsentSerializer::class") || !strings.Contains(model, "DeniedDto? = null") {
			t.Fatal("optional false reference lost missing-versus-null contract", model)
		}
		root := source(t, got, "DeniedDto")
		if !strings.Contains(root, "object AbsentSerializer") || !strings.Contains(root, "descriptor.nullable") {
			t.Fatal("missing strict nullable-descriptor absence codec", root)
		}
	}
}

func TestFalseReferenceFieldPresenceAndOptions(t *testing.T) {
	input := schemas(`{"Holder":{"required":["required","nullable"],"properties":{"optional":{"$ref":"#/components/schemas/Denied"},"required":{"$ref":"#/components/schemas/Denied"},"nullable":{"$ref":"#/components/schemas/Nullable"},"narrow":{"$ref":"#/components/schemas/Nullable","nullable":false}}},"Nullable":{"$ref":"#/components/schemas/Denied","nullable":true},"Denied":false}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for bits := 0; bits < 8; bits++ {
				rq := request(input, "Dto")
				rq.Options.ModelStyle = style
				rq.Options.Validation = checked
				rq.Options.DeduplicateModels = bits&1 != 0
				rq.Options.HoistSharedShapes = bits&2 != 0
				rq.Options.MergeIdenticalModels = bits&4 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				got, e := core.GenerateWithDiagnostics(rq)
				if e != nil {
					t.Fatal(style, checked, bits, e)
				}
				model := source(t, got, "HolderDto")
				if strings.Count(model, "EncodeDefault.Mode.NEVER") != 2 || !strings.Contains(model, "val required: "+rq.Options.PackageName+".models.DeniedDto,") {
					t.Fatal(model)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", e)
				}
			}
		}
	}
}
func TestFalseReferenceFieldOtherRolesStayClosed(t *testing.T) {
	for _, role := range []string{
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied","default":null}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied","readOnly":true}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied","writeOnly":true}}}`,
		`"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}}}}`,
		`"Holder":{"properties":{"value":{"additionalProperties":{"$ref":"#/components/schemas/Denied"}}}}`,
		`"Holder":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Denied"}}}]}`,
		`"Holder":{"properties":{"nested":{"properties":{"value":{"$ref":"#/components/schemas/Denied"}}}}}`,
		`"Holder":{"properties":{"value":false}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied","nullable":"true"}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied","unexpected":1}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Denied":false,`+role+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			// Retain the exact74 inputs as83 explicit inline-container graduations.
			if role == `"Holder":{"properties":{"value":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}}}}` || role == `"Holder":{"properties":{"value":{"additionalProperties":{"$ref":"#/components/schemas/Denied"}}}}` {
				if e != nil {
					t.Fatal("inline collection field graduation", role, e)
				}
				continue
			}
			if role == `"Holder":{"properties":{"value":false}}` {
				if e != nil {
					t.Fatal("exact74literalfalsefield graduation", e)
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("role widened", role, e)
			}
		}
	}
}
func TestFalseReferenceFieldConstructorAndSourceBudgets(t *testing.T) {
	for _, count := range []int{240, 241} {
		props := make([]string, count)
		for i := range props {
			props[i] = fmt.Sprintf(`"p%03d":{"$ref":"#/components/schemas/Denied"}`, i)
		}
		rq := request(schemas(`{"Denied":false,"Holder":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("slot limit lost", e)
		}
	}
	for _, count := range []int{16, 17} {
		properties := make([]string, 240)
		for i := range properties {
			properties[i] = fmt.Sprintf(`"p%03d":{"$ref":"#/components/schemas/Denied"}`, i)
		}
		parts := []string{`"Denied":false`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(properties, ",")))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 16 {
			if e != nil {
				t.Fatal("bounded cumulative capture rejected", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("source work reset", e)
		}
	}
}

func TestFalseReferenceFieldAliasDepth(t *testing.T) {
	for _, count := range []int{46, 47, 48} {
		parts := []string{`"Denied":false`}
		previous := "Denied"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s"}`, name, previous))
			previous = name
		}
		parts = append(parts, fmt.Sprintf(`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/%s"}}}`, previous))
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 46 {
			if e != nil {
				t.Fatal("bounded field alias depth", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("field alias source height lost", count, e)
		}
	}
}
