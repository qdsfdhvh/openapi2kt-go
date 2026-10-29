package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseInlineCollectionFieldMinimal(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}

func TestFalseInlineCollectionFieldProfiles(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"required":["required","nullable"],"properties":{"required":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}},"nullable":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Maybe"}},"mapping":{"additionalProperties":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Maybe","nullable":false}}},"optional":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Maybe"}}}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, checked := range []bool{false, true} {
			for bits := 0; bits < 8; bits++ {
				rq := request(input, "Dto")
				rq.Options.Validation = checked
				rq.Options.ModelStyle = style
				rq.Options.DeduplicateModels = bits&1 != 0
				rq.Options.HoistSharedShapes = bits&2 != 0
				rq.Options.MergeIdenticalModels = bits&4 != 0
				rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
				got, e := core.GenerateWithDiagnostics(rq)
				if e != nil {
					t.Fatal(style, checked, bits, e)
				}
				holder := source(t, got, "HolderDto")
				if strings.Contains(holder, "AbsentSerializer") || !strings.Contains(holder, "kotlin.collections.List<org.example.models.models.BlockedDto>") {
					t.Fatal(holder)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic inline false collections", e)
				}
			}
		}
	}
}
func TestFalseInlineCollectionFieldClosedRoles(t *testing.T) {
	for _, role := range []string{
		`"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}},"default":{}}`,
		`"Holder":{"allOf":[{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}}}]}`,
		`"Holder":{"properties":{"nested":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}}}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":false}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Blocked"}]}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Blocked"}}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","default":null}}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","unknown":true}}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","readOnly":true,"items":{"$ref":"#/components/schemas/Blocked"}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Blocked":false,`+role+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if role == `"Holder":{"properties":{"rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Blocked"}]}}}` {
				if e != nil {
					t.Fatal("original inline tuple reference input", e)
				}
				if !strings.Contains(source(t, got, "HolderDto"), "List<org.example.models.models.BlockedDto>") {
					t.Fatal("inline tuple reference lost actual rejecting root")
				}
				continue
			}
			// Exact83 default[] field input graduates under84 ownannotation proof.
			if role == `"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}}` {
				if e != nil {
					t.Fatal("own inline default graduation", e)
				}
				continue
			}
			if role == `"Holder":{"properties":{"rows":{"type":"array","items":false}}}` {
				text := source(t, got, "HolderDto")
				if e != nil || !strings.Contains(text, "List<org.example.models.models.HolderDto.FalseValue>") || strings.Contains(text, "AbsentSerializer") {
					t.Fatal("original83 literalarrayfield graduation", e, text)
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("inline false collection widened role", role, e)
			}
		}
	}
}
func TestFalseInlineCollectionFieldDepthAndWidth(t *testing.T) {
	for _, count := range []int{46, 47} {
		leaf := `{"$ref":"#/components/schemas/Blocked"}`
		for i := 0; i < count; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{"Blocked":false,"Holder":{"properties":{"rows":`+leaf+`}}}`), "Dto"))
		if count == 46 {
			if e != nil {
				t.Fatal("bounded source", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("field original source depth reset", e)
		}
	}
	for _, count := range []int{240, 241} {
		props := []string{}
		for i := 0; i < count; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, i))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{"Blocked":false,"Holder":{"properties":{`+strings.Join(props, ",")+`}}}`), "Dto"))
		if count == 240 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("JVM240 cap lost", e)
		}
	}
}
func TestFalseInlineCollectionFieldCumulativeWork(t *testing.T) {
	for _, count := range []int{7, 8, 9} {
		props := []string{}
		for i := 0; i < 240; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, i))
		}
		parts := []string{`"Blocked":false`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 7 {
			if e != nil {
				t.Fatal("bounded inline field source work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("inline collection source work reset", e)
		}
	}
}
