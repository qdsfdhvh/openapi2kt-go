package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseDTOConsumingFieldDefaultMinimal(t *testing.T) {
	input := schemas(`{"Blocked":false,"Target":{"properties":{"value":{"$ref":"#/components/schemas/Blocked","nullable":true}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}}`)
	for _, checked := range []bool{false, true} {
		rq := request(input, "Dto")
		rq.Options.Validation = checked
		got, e := core.GenerateWithDiagnostics(rq)
		if e != nil {
			t.Fatal(e)
		}
		model := source(t, got, "ConsumerDto")
		if !strings.Contains(model, "default") || !strings.Contains(model, "value") {
			t.Fatal(model)
		}
	}
}

func TestFalseDTOConsumingFieldDefaultProfiles(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"required":["maybe"],"properties":{"maybe":{"$ref":"#/components/schemas/Maybe"},"blocked":{"$ref":"#/components/schemas/Blocked"},"label":{"type":"string"}}},"Alias":{"$ref":"#/components/schemas/Target"},"Consumer":{"required":["required"],"properties":{"optional":{"$ref":"#/components/schemas/Target","default":{"maybe":null,"label":"synthetic"}},"alias":{"$ref":"#/components/schemas/Alias","default":{"maybe":null}},"required":{"$ref":"#/components/schemas/Target","default":{"maybe":null}},"null":{"$ref":"#/components/schemas/Target","nullable":true,"default":null},"inherited":{"$ref":"#/components/schemas/Target"}}}}`)
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
				model := source(t, got, "ConsumerDto")
				if strings.Contains(source(t, got, "TargetDto"), "public fun default()") || strings.Contains(model, "public val required: org.example.models.models.TargetDto =") || !strings.Contains(model, "public val inherited: org.example.models.models.TargetDto? = null") || !strings.Contains(model, "model default") {
					t.Fatal(model)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic field default", e)
				}
			}
		}
	}
}

func TestFalseDTOConsumingFieldDefaultValues(t *testing.T) {
	for _, tc := range []struct{ field, value, pointer string }{
		{`{"$ref":"#/components/schemas/Maybe"}`, `{"value":true}`, "/default/value"},
		{`{"$ref":"#/components/schemas/Blocked"}`, `{"value":null}`, "/default/value"},
		{`{"$ref":"#/components/schemas/Maybe","nullable":false}`, `{"value":null}`, "/default/value"},
		{`{"$ref":"#/components/schemas/Maybe"}`, `{}`, "/default"},
	} {
		for _, required := range []string{`[]`, `["target"]`} {
			rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"required":["value"],"properties":{"value":`+tc.field+`}},"Consumer":{"required":`+required+`,"properties":{"target":{"$ref":"#/components/schemas/Target","default":`+tc.value+`}}}}`), "Dto")
			got, e := core.GenerateWithDiagnostics(rq)
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, e)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Consumer/properties/target"+tc.pointer {
				t.Fatal(tc, d)
			}
		}
	}
}

func TestFalseDTOConsumingFieldDefaultClosedRoles(t *testing.T) {
	target := `"Target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}}`
	for _, role := range []string{
		`"Consumer":{"properties":{"target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}},"default":{"value":null}}}}`,
		`"Consumer":{"properties":{"targets":{"type":"array","items":{"$ref":"#/components/schemas/Target"},"default":[{"value":null}]}}}`,
		`"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}},"default":{}}`,
		`"Consumer":{"allOf":[{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}]}`,
		`"Outer":{"properties":{"nested":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}}}`,
		`"Alias":{"$ref":"#/components/schemas/Target","default":{"value":null}}`,
		`"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"value":null}}}`,
	} {
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},`+target+`,`+role+`}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if role == `"Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},"default":{"target":{"value":null}}}` {
			if e != nil || !strings.Contains(source(t, got, "OuterDto"), "public fun default()") {
				t.Fatal("original ownerroot member-reference graduation", e)
			}
			continue
		}
		if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("false DTO field role widened", role, e)
		}
	}
}

func TestFalseDTOConsumingFieldDefaultBudgets(t *testing.T) {
	for _, count := range []int{44, 45} {
		parts := []string{`"Blocked":false`}
		prev := "Blocked"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s","nullable":true}`, name, prev))
			prev = name
		}
		parts = append(parts, fmt.Sprintf(`"Target":{"properties":{"value":{"$ref":"#/components/schemas/%s"}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}`, prev))
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 44 {
			if e != nil {
				t.Fatal(e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("source48 cap lost", e)
		}
	}
}

func TestFalseDTOConsumingFieldDefaultCumulativeWork(t *testing.T) {
	for _, count := range []int{8, 9} {
		parts := []string{`"Blocked":false`, `"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true}`, `"Target":{"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}}`}
		props := []string{}
		for i := 0; i < 240; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":{"$ref":"#/components/schemas/Target","default":{"value":null}}`, i))
		}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"C%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 8 {
			if e != nil {
				t.Fatal("bounded consumer source work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("consumer cumulative source work reset", e)
		}
	}
}

// Exact80 target-count refusal input is retained as this explicit81 graduation.
func TestFalseDTOConsumingFieldDefaultTargetCountsGraduation(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Target":{"minProperties":1,"properties":{"value":{"$ref":"#/components/schemas/Maybe"}}},"Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"value":null}}}}}`), "Dto")
	rq.Options.Validation = true
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
	rq.Options.Validation = false
	got, e := core.GenerateWithDiagnostics(rq)
	if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
		t.Fatal("target counts lost validation opt-in", e)
	}
}
