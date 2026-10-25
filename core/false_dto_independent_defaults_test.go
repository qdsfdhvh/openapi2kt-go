package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseDTOIndependentDefaultsMinimal(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"},"label":{"type":"string"}},"default":{"value":null,"label":"root"}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null,"label":"field"}}}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}

func TestFalseDTOIndependentDefaultsOptions(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"},"label":{"type":"string"}},"default":{"value":null,"label":"root"}},"Alias":{"$ref":"#/components/schemas/Target"},"Consumer":{"required":["required"],"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null,"label":"field"}},"alias":{"$ref":"#/components/schemas/Alias","default":{"value":null}},"required":{"$ref":"#/components/schemas/Target","default":{"value":null}},"absent":{"$ref":"#/components/schemas/Target"}}}}`)
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
				target := source(t, got, "TargetDto")
				consumer := source(t, got, "ConsumerDto")
				if !strings.Contains(target, "public fun default()") || !strings.Contains(target, `root`) || !strings.Contains(consumer, `field`) || !strings.Contains(consumer, "public val absent: org.example.models.models.TargetDto? = null") {
					t.Fatal(target, consumer)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("independent annotations nondeterministic", e)
				}
			}
		}
	}
}
func TestFalseDTOIndependentDefaultsInvalidAnnotations(t *testing.T) {
	for _, tc := range []struct{ root, field, pointer string }{
		{`{"value":true}`, `{"value":null}`, "/components/schemas/Target/default/value"},
		{`{"value":null}`, `{"value":true}`, "/components/schemas/Consumer/properties/target/default/value"},
	} {
		for _, required := range []string{`[]`, `["target"]`} {
			rq := request(schemas(fmt.Sprintf(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}},"default":%s},"Consumer":{"required":%s,"properties":{"target":{"$ref":"#/components/schemas/Target","default":%s}}}}`, tc.root, required, tc.field)), "Dto")
			got, e := core.GenerateWithDiagnostics(rq)
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, e)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != tc.pointer {
				t.Fatal(tc, d)
			}
		}
	}
}
