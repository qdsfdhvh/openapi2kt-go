package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseCollectionReferenceMinimal(t *testing.T) {
	input := schemas(`{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}},"Denied":false}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		text := source(t, got, "RowsDto")
		if !strings.Contains(text, "kotlin.collections.List<"+rq.Options.PackageName+".models.DeniedDto>") {
			t.Fatal(text)
		}
	}
}

func TestFalseCollectionReferenceOptionsAndRoles(t *testing.T) {
	input := schemas(`{"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Nullable"}},"Mapping":{"additionalProperties":{"$ref":"#/components/schemas/Blocked"}},"Nullable":{"$ref":"#/components/schemas/Blocked","nullable":true},"Blocked":false,"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}}`)
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
				if !strings.Contains(source(t, got, "RowsDto"), "BlockedDto?>") || strings.Contains(source(t, got, "BlockedDto"), "AbsentSerializer") {
					t.Fatal("lost element codec/nullability or invented absence helper")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic", e)
				}
			}
		}
	}
	for _, role := range []string{
		`"Rows":{"type":"array","items":false}`,
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","default":null}}`,
		`"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Blocked"}]}`,
		`"Rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Blocked"}}`,
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","unknown":true}}`,
		`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","nullable":"true"}}`,
		`"Rows":{"properties":{"fixed":{"type":"string"}},"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}}`,
		`"Rows":{"type":"array","items":{"properties":{"nested":{"$ref":"#/components/schemas/Blocked"}}}}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Blocked":false,`+role+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if role == `"Rows":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Blocked"}]}` {
				if e != nil {
					t.Fatal("original named tuple reference input", e)
				}
				if !strings.Contains(source(t, got, "RowsDto"), "List<org.example.models.models.BlockedDto>") {
					t.Fatal("named tuple reference lost actual rejecting root")
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("unrelated false role admitted", role, e)
			}
		}
	}
}
func TestFalseCollectionReferenceSourceBudgets(t *testing.T) {
	for _, count := range []int{47, 48} {
		leaf := `{"$ref":"#/components/schemas/Blocked"}`
		for i := 0; i < count; i++ {
			leaf = `{"type":"array","items":` + leaf + `}`
		}
		rq := request(schemas(`{"Blocked":false,"Rows":`+leaf+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 47 {
			if e != nil {
				t.Fatal("bounded original nested source", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("source depth reset", e)
		}
	}
	for _, count := range []int{1800, 1820} {
		parts := []string{`"Blocked":false`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"R%04d":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, i))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 1800 {
			if e != nil {
				t.Fatal("bounded original cumulative work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("cumulative source work reset", e)
		}
	}
}

func TestFalseCollectionReferenceCachedIncomingDepth(t *testing.T) {
	for _, count := range []int{45, 46} {
		for _, forward := range []bool{false, true} {
			leaf := `{"$ref":"#/components/schemas/Blocked"}`
			for i := 0; i < count; i++ {
				leaf = `{"type":"array","items":` + leaf + `}`
			}
			entries := []string{`"Blocked":false`, `"Rows":` + leaf, `"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}}}`}
			if forward {
				entries[0], entries[2] = entries[2], entries[0]
			}
			rq := request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")
			got, e := core.GenerateWithDiagnostics(rq)
			if count == 45 {
				if e != nil {
					t.Fatal("bounded incoming reference source", e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("cached false root hop lost", forward, e)
			}
		}
	}
}

func TestFalseCollectionReferenceBoundsKeepValidationGate(t *testing.T) {
	input := schemas(`{"Blocked":false,"Rows":{"type":"array","maxItems":0,"items":{"$ref":"#/components/schemas/Blocked"}},"Map":{"additionalProperties":{"$ref":"#/components/schemas/Blocked"},"minProperties":0},"Holder":{"required":["rows","mapping"],"properties":{"rows":{"$ref":"#/components/schemas/Rows"},"mapping":{"$ref":"#/components/schemas/Map"}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if checked {
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(source(t, got, "HolderDto"), "require(") {
				t.Fatal("lost actual typed constructor counts")
			}
		} else if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("unchecked count admission widened", e)
		}
	}
}

// The former collection-own default rejection retains its exact source input.
func TestFalseCollectionOriginalOwnDefaultGraduation(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Rows":{"type":"array","default":[],"items":{"$ref":"#/components/schemas/Blocked"}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}
