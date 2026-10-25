package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseCollectionOwnDefaultMinimal(t *testing.T) {
	input := schemas(`{"Blocked":false,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		model := source(t, got, "RowsDto")
		if !strings.Contains(model, "RowsDto default") || !strings.Contains(model, "decodeFromString") {
			t.Fatal(model)
		}
	}
}

func TestFalseCollectionOwnDefaultProfilesAndNoInheritance(t *testing.T) {
	input := schemas(`{"Blocked":false,"Nullable":{"$ref":"#/components/schemas/Blocked","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]},"Mapping":{"additionalProperties":{"$ref":"#/components/schemas/Blocked"},"default":{}},"Nulls":{"type":"array","items":{"$ref":"#/components/schemas/Nullable"},"default":[null,null]},"Nested":{"additionalProperties":{"type":"array","items":{"$ref":"#/components/schemas/Nullable"}},"default":{"description":[null]}},"Maybe":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Blocked"},"default":null},"Alias":{"$ref":"#/components/schemas/Rows"},"Holder":{"required":["rows"],"properties":{"rows":{"$ref":"#/components/schemas/Rows"},"maybe":{"$ref":"#/components/schemas/Maybe"}}}}`)
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
				for _, name := range []string{"RowsDto", "MappingDto", "NullsDto", "NestedDto", "MaybeDto"} {
					if !strings.Contains(source(t, got, name), name+" default") {
						t.Fatal("missing own factory", name)
					}
				}
				if strings.Contains(source(t, got, "AliasDto"), "AliasDto default") || strings.Contains(source(t, got, "HolderDto"), "openapi-kt default") {
					t.Fatal("target default inherited")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic own factory", e)
				}
			}
		}
	}
}
func TestFalseCollectionOwnDefaultInvalidValuesAndClosedRoles(t *testing.T) {
	for _, spec := range []struct{ schema, pointer string }{
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[null]}`, "/default/0"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[1]}`, "/default/0"},
		{`{"additionalProperties":{"$ref":"#/components/schemas/Blocked"},"default":{"key":null}}`, "/default/key"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Nullable"},"default":[{}]}`, "/default/0"},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":null}`, "/default"},
	} {
		rq := request(schemas(`{"Blocked":false,"Nullable":{"$ref":"#/components/schemas/Blocked","nullable":true},"Rows":`+spec.schema+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("invalid false default admitted", spec, e)
		}
		diag := e.(core.Diagnostics)[0]
		if diag.Code != "E_SCHEMA_INVALID" || diag.Pointer == nil || *diag.Pointer != "/components/schemas/Rows"+spec.pointer {
			t.Fatal("lost default source Pointer", spec, diag)
		}
	}
	for _, role := range []string{
		`"Alias":{"$ref":"#/components/schemas/Rows","default":[]}`,
		`"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows","default":[]}}}`,
		`"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}}`,
		`"Holder":{"properties":{"rows":{"$ref":"#/components/schemas/Rows"}},"default":{"rows":[]}}`,
		`"Bad":{"type":"array","items":{"$ref":"#/components/schemas/Blocked","default":null},"default":[]}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]},`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if role == `"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}}` {
			if e != nil {
				t.Fatal("exact76 owninlinefielddefault graduation", e)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("root-own factory widened unrelated role", role, e)
		}
	}
}

func TestFalseCollectionOwnDefaultByteBudget(t *testing.T) {
	for _, count := range []int{6553, 6554} {
		values := strings.TrimSuffix(strings.Repeat("null,", count), ",")
		input := schemas(byteStringForFalseDefault(values))
		rq := request(input, "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 6553 {
			if e != nil {
				t.Fatal("bounded32766byte default rejected", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("default32768byte cap widened", e)
		}
	}
}
func byteStringForFalseDefault(values string) string {
	return fmt.Sprintf(`{"Blocked":false,"Nullable":{"$ref":"#/components/schemas/Blocked","nullable":true},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Nullable"},"default":[%s]}}`, values)
}
