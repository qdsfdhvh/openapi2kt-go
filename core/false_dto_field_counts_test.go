package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseDTOFieldTargetCountsMinimal(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"minProperties":1,"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}}`), "Dto")
	rq.Options.Validation = true
	got, e := core.GenerateWithDiagnostics(rq)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(source(t, got, "ConsumerDto"), "model default") {
		t.Fatal(source(t, got, "ConsumerDto"))
	}
}

func TestFalseDTOFieldTargetCountValues(t *testing.T) {
	for _, tc := range []struct {
		bounds, value string
		ok            bool
		pointer       string
	}{
		{`"minProperties":1,"maxProperties":1`, `{"value":null}`, true, ""},
		{`"minProperties":1`, `{}`, false, "/components/schemas/Consumer/properties/target/default"},
		{`"maxProperties":0`, `{"value":null}`, false, "/components/schemas/Consumer/properties/target/default"},
		{`"minProperties":2,"maxProperties":1`, `{"value":null}`, false, "/components/schemas/Target/maxProperties"},
		{`"minProperties":-1`, `{"value":null}`, false, "/components/schemas/Target/minProperties"},
		{`"minProperties":true`, `{"value":null}`, false, "/components/schemas/Target/minProperties"},
		{`"maxProperties":2147483648`, `{"value":null}`, false, "/components/schemas/Target/maxProperties"},
	} {
		for _, req := range []string{`[]`, `["target"]`} {
			rq := request(schemas(fmt.Sprintf(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{%s,"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Consumer":{"required":%s,"properties":{"target":{"$ref":"#/components/schemas/Target","default":%s}}}}`, tc.bounds, req, tc.value)), "Dto")
			rq.Options.Validation = true
			got, e := core.GenerateWithDiagnostics(rq)
			if tc.ok {
				if e != nil {
					t.Fatal(tc, e)
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal("invalidcount admitted", tc, e)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != tc.pointer {
				t.Fatal(tc, d)
			}
		}
	}
}

func TestFalseDTOFieldTargetCountScope(t *testing.T) {
	for _, role := range []string{
		`"Alias":{"$ref":"#/components/schemas/Target","default":{"value":null}}`,
		`"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"value":null}}}`,
		`"Outer":{"properties":{"targets":{"type":"array","items":{"$ref":"#/components/schemas/Target"},"default":[{"value":null}]}}}`,
		`"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}},"default":{}}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"minProperties":1,"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},`+role+`}`), "Dto")
		rq.Options.Validation = true
		got, e := core.GenerateWithDiagnostics(rq)
		if role == `"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"value":null}}}` {
			if e != nil || !strings.Contains(source(t, got, "OuterDto"), "public fun default()") {
				t.Fatal("original ownerroot member-reference graduation", e)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("counts grant leaked", role, e)
		}
	}
}

func TestFalseDTOFieldTargetCountOptions(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"required":["value"],"minProperties":1,"maxProperties":2,"properties":{"value":{"$ref":"#/components/schemas/Maybe"},"blocked":{"$ref":"#/components/schemas/Blocked"}}},"Alias":{"$ref":"#/components/schemas/Target"},"NullableTarget":{"nullable":true,"minProperties":1,"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Consumer":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/Target","default":{"value":null}},"alias":{"$ref":"#/components/schemas/Alias","default":{"value":null}},"required":{"$ref":"#/components/schemas/Target","default":{"value":null}},"nullable":{"$ref":"#/components/schemas/NullableTarget","default":null}}}}`)
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for bits := 0; bits < 8; bits++ {
			rq := request(input, "Dto")
			rq.Options.Validation = true
			rq.Options.ModelStyle = style
			rq.Options.DeduplicateModels = bits&1 != 0
			rq.Options.HoistSharedShapes = bits&2 != 0
			rq.Options.MergeIdenticalModels = bits&4 != 0
			rq.Options.LockHoistedNames = rq.Options.HoistSharedShapes
			got, e := core.GenerateWithDiagnostics(rq)
			if e != nil {
				t.Fatal(style, bits, e)
			}
			again, e := core.GenerateWithDiagnostics(rq)
			if e != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("counted defaults nondeterministic", e)
			}
			rq.Options.Validation = false
			failed, e := core.GenerateWithDiagnostics(rq)
			if e == nil || !reflect.DeepEqual(failed, core.GenerationReport{}) {
				t.Fatal("counted defaults without opt-in", e)
			}
		}
	}
}
