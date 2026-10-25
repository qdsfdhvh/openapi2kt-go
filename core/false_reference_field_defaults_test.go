package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseReferenceFieldOwnNullDefaultMinimal(t *testing.T) {
	input := schemas(`{"Blocked":false,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		model := source(t, got, "HolderDto")
		if !strings.Contains(model, "BlockedDto? = null") || strings.Contains(model, "AbsentSerializer") {
			t.Fatal(model)
		}
	}
}

func TestFalseReferenceOwnFieldDefaultOptionsAndRequiredness(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false},"Holder":{"required":["required"],"properties":{"required":{"$ref":"#/components/schemas/Maybe","default":null},"optional":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null},"targetOnly":{"$ref":"#/components/schemas/Maybe"},"narrow":{"$ref":"#/components/schemas/Maybe","nullable":false}}}}`)
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
				required := "public val required: " + rq.Options.PackageName + ".models.BlockedDto?,"
				if !strings.Contains(model, required) || strings.Count(model, "EncodeDefault.Mode.NEVER") != 1 || strings.Contains(model, "MaybeDto default") {
					t.Fatal("required/default/strictoptional identity lost", model)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic own field default", e)
				}
			}
		}
	}
}
func TestFalseReferenceOwnFieldDefaultInvalidValuesAndClosedRoles(t *testing.T) {
	for _, value := range []string{`true`, `0`, `"text"`, `[]`, `{}`} {
		rq := request(schemas(`{"Blocked":false,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":`+value+`}}}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("nonnull own field default admitted", value, e)
		}
		d := e.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder/properties/value/default" {
			t.Fatal("lost original field default Pointer", d)
		}
	}
	for _, role := range []string{
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","default":null}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Maybe","nullable":false,"default":null}}}`,
		`"Holder":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Maybe","default":null}}}]}`,
		`"Holder":{"properties":{"nested":{"properties":{"value":{"$ref":"#/components/schemas/Maybe","default":null}}}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Maybe","default":null,"readOnly":true}}}`,
		`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Maybe","default":null,"writeOnly":true}}}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("own field role widened", role, e)
		}
	}
}
func TestFalseReferenceOwnFieldDefaultSourceBudgets(t *testing.T) {
	for _, count := range []int{46, 47} {
		parts := []string{`"Blocked":false`}
		previous := "Blocked"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s","nullable":true}`, name, previous))
			previous = name
		}
		parts = append(parts, fmt.Sprintf(`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/%s","default":null}}}`, previous))
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 46 {
			if e != nil {
				t.Fatal("bounded original alias field source", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("owned field alias depth lost", e)
		}
	}
	for _, count := range []int{11, 12} {
		props := make([]string, 240)
		for i := range props {
			props[i] = fmt.Sprintf(`"p%03d":{"$ref":"#/components/schemas/Maybe","default":null}`, i)
		}
		parts := []string{`"Blocked":false`, `"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true}`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 11 {
			if e != nil {
				t.Fatal("bounded cumulative owned fields", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("owned field sourcework reset", e)
		}
	}
}

func TestFalseObjectMemberOriginalFieldDefaultGraduation(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}},"default":{"value":null}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}
