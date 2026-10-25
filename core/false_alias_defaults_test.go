package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseAliasOwnNullFactoryMinimal(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		model := source(t, got, "MaybeDto")
		if !strings.Contains(model, "MaybeDto default") || !strings.Contains(model, "decodeFromString") {
			t.Fatal(model)
		}
	}
}

func TestFalseAliasOwnDefaultOptionsAndIndependentIntent(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"OwnInherited":{"$ref":"#/components/schemas/Maybe","default":null},"Inherited":{"$ref":"#/components/schemas/Maybe"},"Narrow":{"$ref":"#/components/schemas/Maybe","nullable":false},"Again":{"$ref":"#/components/schemas/Narrow","nullable":true,"default":null},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"},"default":[null]},"Holder":{"properties":{"maybe":{"$ref":"#/components/schemas/Maybe"}}}}`)
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
				for _, name := range []string{"MaybeDto", "OwnInheritedDto", "AgainDto", "RowsDto"} {
					if !strings.Contains(source(t, got, name), name+" default") {
						t.Fatal("missing own default", name)
					}
				}
				for _, name := range []string{"InheritedDto", "NarrowDto", "BlockedDto"} {
					if strings.Contains(source(t, got, name), name+" default") {
						t.Fatal("inherited default", name)
					}
				}
				if strings.Contains(source(t, got, "HolderDto"), "MaybeDto default") {
					t.Fatal("factory inherited by field")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic own alias factory", e)
				}
			}
		}
	}
}
func TestFalseAliasOwnDefaultInvalidValuesAndClosedRoles(t *testing.T) {
	for _, value := range []string{`true`, `0`, `"text"`, `[]`, `{}`} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":`+value+`}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("nonnull false alias default admitted", value, e)
		}
		d := e.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Maybe/default" {
			t.Fatal("lost original default Pointer", d)
		}
	}
	for _, role := range []string{
		`"Bad":{"$ref":"#/components/schemas/Maybe","nullable":false,"default":null}`,
		`"Bad":{"$ref":"#/components/schemas/Blocked","default":null}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("alias-own default widened unrelated role", role, e)
		}
	}
}
func TestFalseAliasOwnDefaultSourceBudgets(t *testing.T) {
	for _, count := range []int{48, 49} {
		parts := []string{`"Blocked":false`}
		previous := "Blocked"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s","nullable":true,"default":null}`, name, previous))
			previous = name
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 48 {
			if e != nil {
				t.Fatal("bounded original alias source", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("original alias depth lost", e)
		}
	}
	for _, count := range []int{2700, 2800} {
		parts := []string{`"Blocked":false`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"A%04d":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null}`, i))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 2700 {
			if e != nil {
				t.Fatal("bounded cumulative original alias sources", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("original cumulative source work lost", e)
		}
	}
}

func TestFalseAliasOriginalOwnFieldNullDefaultGraduation(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"Holder":{"properties":{"maybe":{"$ref":"#/components/schemas/Maybe","default":null}}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}

func TestFalseObjectMemberOriginalAliasDefaultGraduation(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"Holder":{"properties":{"maybe":{"$ref":"#/components/schemas/Maybe"}},"default":{"maybe":null}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}
