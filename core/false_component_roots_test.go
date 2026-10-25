package core_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestFalseComponentRootsMinimal(t *testing.T) {
	input := schemas(`{"Denied":false,"Alias":{"$ref":"#/components/schemas/Denied"},"Nullable":{"$ref":"#/components/schemas/Alias","nullable":true},"Narrow":{"$ref":"#/components/schemas/Nullable","nullable":false}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(checked, err)
		}
		if !strings.Contains(source(t, got, "DeniedDto"), "sealed class DeniedDto private constructor()") {
			t.Fatal("false root acquired an inhabitable/raw carrier", source(t, got, "DeniedDto"))
		}
		for name, tail := range map[string]string{"AliasDto": "", "NullableDto": "?", "NarrowDto": ""} {
			if !strings.Contains(source(t, got, name), "typealias "+name+" = "+rq.Options.PackageName+".models.DeniedDto"+tail+"\n") {
				t.Fatal("lost canonical false codec or nullable intent", name, source(t, got, name))
			}
		}
	}
}

func TestFalseComponentRootOptionsStylesAndRefusalRoles(t *testing.T) {
	input := schemas(`{"Denied":false,"Peer":false,"Alias":{"$ref":"#/components/schemas/Denied"},"Nullable":{"$ref":"#/components/schemas/Alias","nullable":true}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, suffix := range []string{"", "Dto"} {
			for _, checked := range []bool{false, true} {
				for bits := 0; bits < 8; bits++ {
					rq := request(input, suffix)
					rq.Options.ModelStyle = style
					rq.Options.Validation = checked
					rq.Options.DeduplicateModels = bits&1 != 0
					rq.Options.HoistSharedShapes = bits&2 != 0
					rq.Options.MergeIdenticalModels = bits&4 != 0
					rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
					got, e := core.GenerateWithDiagnostics(rq)
					if e != nil {
						t.Fatal(style, suffix, checked, bits, e)
					}
					if !strings.Contains(source(t, got, "Denied"+suffix), "sealed class Denied"+suffix+" private constructor()") || !strings.Contains(source(t, got, "Peer"+suffix), "sealed class Peer"+suffix+" private constructor()") {
						t.Fatal("false roots structurally collapsed")
					}
					again, retry := core.GenerateWithDiagnostics(rq)
					if retry != nil || !reflect.DeepEqual(got, again) || !bytes.Equal(input, rq.Document) {
						t.Fatal("false root nondeterminism", retry)
					}
				}
			}
		}
	}
	for _, role := range []string{
		`"Choice":{"oneOf":[{"$ref":"#/components/schemas/Denied"},{"type":"string"}]}`,
		`"All":{"allOf":[{"$ref":"#/components/schemas/Denied"}]}`,
		`"Alias":{"$ref":"#/components/schemas/Denied","default":null}`,
	} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Denied":false,`+role+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("false root widened unrelated role", role, checked, e)
			}
		}
	}
}

func TestFalseComponentRootAliasDepth(t *testing.T) {
	for _, count := range []int{48, 49} {
		entries := []string{`"Denied":false`}
		previous := "Denied"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			entries = append(entries, `"`+name+`":{"$ref":"#/components/schemas/`+previous+`","nullable":true}`)
			previous = name
		}
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")
			rq.Options.Validation = checked
			got, e := core.GenerateWithDiagnostics(rq)
			if count == 48 {
				if e != nil {
					t.Fatal("valid false alias source depth", e)
				}
			} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("false aliases erased source depth", count, e)
			}
		}
	}
}

func TestFalseComponentRootCumulativeSourceWork(t *testing.T) {
	for _, count := range []int{8000, 8200} {
		entries := make([]string, count)
		for i := range entries {
			entries[i] = fmt.Sprintf(`"R%05d":false`, i)
		}
		rq := request(schemas(`{`+strings.Join(entries, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 8000 {
			if e != nil || len(got.Artifacts.Files) != count {
				t.Fatal("bounded false root source lost", count, e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("false root captures reset cumulative work", e)
		}
	}
}

// Preserve the exact former field rejection as an explicit graduation control.
func TestFalseComponentRootOriginalFieldGraduation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		rq := request(schemas(`{"Denied":false,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Denied"}}}}`), "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil || !strings.Contains(source(t, got, "HolderDto"), "DeniedDto.AbsentSerializer::class") {
			t.Fatal(e)
		}
	}
}

func TestFalseComponentRootOriginalCollectionGraduation(t *testing.T) {
	for _, role := range []string{`"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}}`, `"Mapping":{"type":"object","additionalProperties":{"$ref":"#/components/schemas/Denied"}}`} {
		for _, checked := range []bool{false, true} {
			rq := request(schemas(`{"Denied":false,`+role+`}`), "Dto")
			rq.Options.Validation = checked
			if _, e := core.GenerateWithDiagnostics(rq); e != nil {
				t.Fatal("original source collection graduation", role, e)
			}
		}
	}
}
