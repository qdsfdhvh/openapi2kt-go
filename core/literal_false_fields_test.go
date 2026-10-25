package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestLiteralFalseFieldMinimal(t *testing.T) {
	rq := request(schemas(`{"Holder":{"properties":{"blocked":false}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}

func TestLiteralFalseFieldProfilesAndNaming(t *testing.T) {
	input := schemas(`{"Holder":{"required":["required"],"properties":{"optional":false,"required":false,"falseValue":{"properties":{"label":{"type":"string"}}},"serializer":false,"kotlin":false,"kotlinx":false}},"FalseValue":{"properties":{"value":false}},"RequiredOnly":{"required":["value"],"properties":{"value":false}}}`)
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
				holder := source(t, got, "HolderDto")
				if !strings.Contains(holder, "sealed class FalseValue2 private constructor()") || strings.Count(holder, "sealed class FalseValue2") != 1 || !strings.Contains(holder, "descriptors.nullable") {
					t.Fatal(holder)
				}
				only := source(t, got, "RequiredOnlyDto")
				if strings.Contains(only, "AbsentSerializer") || strings.Contains(only, "descriptors.nullable") {
					t.Fatal("unusedabsencehelper emitted", only)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("literalfalsefield nondeterminism", e)
				}
			}
		}
	}
}
func TestLiteralFalseFieldExcludedContexts(t *testing.T) {
	for _, role := range []string{
		`"Holder":{"properties":{"nested":{"properties":{"value":false}}}}`,
		`"Holder":{"allOf":[{"properties":{"value":false}}]}`,
		`"Holder":{"properties":{"value":false},"default":{}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false}}}`,
		`"Holder":{"properties":{"value":{"oneOf":[false,{"type":"null"}]}}}`,
	} {
		rq := request(schemas(`{`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if role == `"Holder":{"properties":{"value":false},"default":{}}` {
			if e != nil || !strings.Contains(source(t, got, "HolderDto"), "public fun default()") {
				t.Fatal("original root-own literalfalse default graduation", e)
			}
			continue
		}
		if role == `"Holder":{"properties":{"rows":{"type":"array","items":false}}}` {
			if e != nil || !strings.Contains(source(t, got, "HolderDto"), "List<org.example.models.models.HolderDto.FalseValue>") {
				t.Fatal("original85 literalarrayfield graduation", e)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("literalfieldrole leaked", role, e)
		}
	}
	rq := request([]byte(`{"openapi":"3.0.3","components":{"schemas":{"Holder":{"properties":{"value":false}}}}}`), "Dto")
	got, e := core.GenerateWithDiagnostics(rq)
	if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("3.0Booleanprofile widened", e)
	}
}
func TestLiteralFalseFieldJVMWidth(t *testing.T) {
	for _, count := range []int{240, 241} {
		props := []string{}
		for i := 0; i < count; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":false`, i))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{"Wide":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto"))
		if count == 240 {
			if e != nil {
				t.Fatal(e)
			}
			if strings.Count(source(t, got, "WideDto"), "sealed class FalseValue") != 1 {
				t.Fatal("literalcodec notsharedinactualowner")
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("JVM240caplost", e)
		}
	}
}
func TestLiteralFalseFieldsCumulativeSourceWork(t *testing.T) {
	for _, count := range []int{33, 34} {
		props := []string{}
		for i := 0; i < 240; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":false`, i))
		}
		parts := []string{}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 33 {
			if e != nil {
				t.Fatal("bounded literal source work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("literal source budget reset", e)
		}
	}
}
func TestFalseSourceBudgetForwardDeclaration(t *testing.T) {
	for _, forward := range []bool{false, true} {
		for _, count := range []int{7, 8} {
			rootName := "Blocked"
			if forward {
				rootName = "ZBlocked"
			}
			props := []string{}
			for i := 0; i < 240; i++ {
				props = append(props, fmt.Sprintf(`"p%03d":{"type":"array","items":{"$ref":"#/components/schemas/%s"}}`, i, rootName))
			}
			parts := []string{}
			for i := 0; i < count; i++ {
				parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
			}
			if forward {
				parts = append(parts, fmt.Sprintf(`"%s":false`, rootName))
			} else {
				parts = append([]string{fmt.Sprintf(`"%s":false`, rootName)}, parts...)
			}
			got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
			if count == 7 {
				if e != nil {
					t.Fatal("bounded originalfalsework", forward, e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("declarationorder changedcumulativefalsebudget", forward, e)
			}
		}
	}
}
