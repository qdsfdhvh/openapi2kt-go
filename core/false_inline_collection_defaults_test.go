package core_test

import (
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func TestFalseInlineCollectionOwnDefaultMinimal(t *testing.T) {
	rq := request(schemas(`{"Blocked":false,"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}}}}`), "Dto")
	if _, e := core.GenerateWithDiagnostics(rq); e != nil {
		t.Fatal(e)
	}
}

func TestFalseInlineCollectionOwnDefaultOptions(t *testing.T) {
	input := schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"required":["required","requiredNull"],"properties":{"empty":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]},"mapping":{"additionalProperties":{"$ref":"#/components/schemas/Maybe"},"default":{"description":null,"$ref":null}},"nested":{"type":"array","items":{"additionalProperties":{"$ref":"#/components/schemas/Maybe"}},"default":[{"k":null},{}]},"required":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"},"default":[null]},"nullable":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Blocked"},"default":null},"requiredNull":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Blocked"},"default":null}}}}`)
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
				if !strings.Contains(holder, "default tree") || strings.Contains(holder, "public val required: kotlin.collections.List<org.example.models.models.BlockedDto?> =") {
					t.Fatal(holder)
				}
				again, e := core.GenerateWithDiagnostics(rq)
				if e != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("inlineownliteral nondeterminism", e)
				}
			}
		}
	}
}
func TestFalseInlineCollectionOwnDefaultValues(t *testing.T) {
	for _, tc := range []struct {
		schema, value, pointer string
		ok                     bool
	}{
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, `[]`, "", true},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, `[null]`, "/default/0", false},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}}`, `[null]`, "", true},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Maybe"}}`, `[1]`, "/default/0", false},
		{`{"additionalProperties":{"$ref":"#/components/schemas/Blocked"}}`, `{"k":null}`, "/default/k", false},
		{`{"type":"array","items":{"$ref":"#/components/schemas/Blocked"}}`, `null`, "/default", false},
		{`{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Blocked"}}`, `null`, "", true},
	} {
		for _, required := range []string{`[]`, `["rows"]`} {
			schema := strings.TrimSuffix(tc.schema, "}") + `,"default":` + tc.value + `}`
			rq := request(schemas(fmt.Sprintf(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"required":%s,"properties":{"rows":%s}}}`, required, schema)), "Dto")
			got, e := core.GenerateWithDiagnostics(rq)
			if tc.ok {
				if e != nil {
					t.Fatal(tc, e)
				}
				continue
			}
			if e == nil || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(tc, e)
			}
			d := e.(core.Diagnostics)[0]
			if d.Code != "E_SCHEMA_INVALID" || d.Pointer == nil || *d.Pointer != "/components/schemas/Holder/properties/rows"+tc.pointer {
				t.Fatal(tc, d)
			}
		}
	}
}
func TestFalseInlineCollectionOwnDefaultByteBudget(t *testing.T) {
	for _, count := range []int{6553, 6554} {
		value := `[` + strings.TrimSuffix(strings.Repeat("null,", count), ",") + `]`
		rq := request(schemas(`{"Blocked":false,"Maybe":{"$ref":"#/components/schemas/Blocked","nullable":true},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Maybe"},"default":`+value+`}}}}`), "Dto")
		got, e := core.GenerateWithDiagnostics(rq)
		if count == 6553 {
			if e != nil {
				t.Fatal("32766canonicalbytes", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("32KiB cap lost", e)
		}
	}
}
func TestFalseInlineCollectionOwnDefaultSourceBudgets(t *testing.T) {
	for _, count := range []int{44, 45} {
		parts := []string{`"Blocked":false`}
		previous := "Blocked"
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("A%02d", i)
			parts = append(parts, fmt.Sprintf(`"%s":{"$ref":"#/components/schemas/%s","nullable":false}`, name, previous))
			previous = name
		}
		parts = append(parts, fmt.Sprintf(`"Alias":{"$ref":"#/components/schemas/%s","nullable":true},"Holder":{"properties":{"rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias"},"default":[null]}}}`, previous))
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 44 {
			if e != nil {
				t.Fatal("bounded own-default source alias", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("default source depth reset", e)
		}
	}
	for _, count := range []int{6, 7} {
		props := []string{}
		for i := 0; i < 240; i++ {
			props = append(props, fmt.Sprintf(`"p%03d":{"type":"array","items":{"$ref":"#/components/schemas/Blocked"},"default":[]}`, i))
		}
		parts := []string{`"Blocked":false`}
		for i := 0; i < count; i++ {
			parts = append(parts, fmt.Sprintf(`"H%02d":{"properties":{%s}}`, i, strings.Join(props, ",")))
		}
		got, e := core.GenerateWithDiagnostics(request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto"))
		if count == 6 {
			if e != nil {
				t.Fatal("bounded fieldliteral source work", e)
			}
		} else if e == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{}) {
			t.Fatal("fieldliteral cumulative source reset", e)
		}
	}
}
