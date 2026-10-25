package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseObjectRootDefaultMemberMinimal(t *testing.T) {
	input := schemas(`{"Blocked":false,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","nullable":true}},"default":{"value":null}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(source(t, got, "HolderDto"), "public fun default()") {
			t.Fatal(source(t, got, "HolderDto"))
		}
	}
}

func TestFalseObjectRootDefaultMemberOptionsAndPresence(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true,"default":null},"OptionalOnly":{"properties":{"blocked":{"$ref":"#/components/schemas/Blocked"}},"default":{}},"Nullable":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}},"default":{"value":null}},"Required":{"required":["value"],"properties":{"value":{"$ref":"#/components/schemas/Maybe","default":null}},"default":{"value":null}},"RootNullable":{"type":"object","nullable":true,"properties":{"blocked":{"$ref":"#/components/schemas/Blocked"}},"default":null},"Alias":{"$ref":"#/components/schemas/Nullable"}}`)
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
				for _, name := range []string{"OptionalOnlyDto", "NullableDto", "RequiredDto", "RootNullableDto"} {
					if !strings.Contains(source(t, got, name), "public fun default()") {
						t.Fatal("missing own model factory", name)
					}
				}
				if strings.Contains(source(t, got, "AliasDto"), "AliasDto default") {
					t.Fatal("target factory inherited")
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic own object default", e)
				}
			}
		}
	}
}
func TestFalseObjectRootDefaultMemberInvalidValuesAndClosedRoles(t *testing.T) {
	for _, tc := range []struct{ properties, required, value, pointer string }{
		{`{"value":{"$ref":"#/components/schemas/Blocked"}}`, `[]`, `{"value":null}`, "/default/value"},
		{`{"value":{"$ref":"#/components/schemas/Maybe"}}`, `[]`, `{"value":1}`, "/default/value"},
		{`{"value":{"$ref":"#/components/schemas/Maybe"}}`, `["value"]`, `{}`, "/default"},
		{`{"value":{"$ref":"#/components/schemas/Maybe","nullable":false}}`, `[]`, `{"value":null}`, "/default/value"},
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"required":`+tc.required+`,"properties":`+tc.properties+`,"default":`+tc.value+`}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("invalid false object literal admitted", tc, e)
		}
		d := e.(core.Diagnostics)[0]
		if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder"+tc.pointer {
			t.Fatal("lost original default value Pointer", tc, d)
		}
	}
	for _, role := range []string{
		`"Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Holder","default":{"value":null}}}}`,
		`"Alias":{"$ref":"#/components/schemas/Holder","default":{"value":null}}`,
		`"Outer":{"properties":{"nested":{"$ref":"#/components/schemas/Holder"}},"default":{"nested":{"value":null}}}`,
		`"All":{"allOf":[{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}}],"default":{"value":null}}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}},"default":{"value":null}},`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		// Exact79 consuming-field input graduates only with82 independent
		// actual target-factory authority; the other roles remain rejected.
		if strings.HasPrefix(role, `"Consumer":`) {
			if e != nil {
				t.Fatal("independent consuming/target default", e)
			}
			continue
		}
		if role == `"Outer":{"properties":{"nested":{"$ref":"#/components/schemas/Holder"}},"default":{"nested":{"value":null}}}` {
			if e != nil || !strings.Contains(source(t, got, "OuterDto"), "public fun default()") {
				t.Fatal("original ownerroot member-reference graduation", e)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("root-own object default widened other role", role, e)
		}
	}
}

func TestFalseObjectRootDefaultMemberSourceBudgets(t *testing.T) {
	for _, count := range []int{46, 47} {
		parts := []string{`"Blocked":false`}
		previous := "Blocked"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s","nullable":true}`, name, previous))
			previous = name
		}
		parts = append(parts, fmt.Sprintf(`"Holder":{"properties":{"value":{"$ref":"#/components/schemas/%s"}},"default":{"value":null}}`, previous))
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 46 {
			if e != nil {
				t.Fatal("bounded original default-member source", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("default member source hops lost", e)
		}
	}
	for _, count := range []int{8, 9} {
		props := make([]string, 240)
		values := make([]string, 240)
		for i := range props {
			props[i] = fmt.Sprintf(`"p%03d":{"$ref":"#/components/schemas/Maybe","default":null}`, i)
			values[i] = fmt.Sprintf(`"p%03d":null`, i)
		}
		parts := []string{`"Blocked":false`, `"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true}`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s},"default":{%s}}`, i, strings.Join(props, ","), strings.Join(values, ",")))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 8 {
			if e != nil {
				t.Fatal("bounded original default source work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("original default source capture reset", e)
		}
	}
}
