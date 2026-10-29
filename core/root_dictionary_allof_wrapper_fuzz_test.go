package core_test

import (
	"fmt"
	"reflect"
	"testing"

	"openapi-kt/core"
)

// A dictionary intersection with only neutral siblings has the same
// generated view as the corresponding pure alias and nullable intent.
func FuzzRootDictionaryAllOfWrapperMatchesAlias(f *testing.F) {
	f.Add(uint8(0), false)
	f.Add(uint8(17), true)
	f.Add(uint8(63), true)
	f.Add(uint8(128), false)
	f.Add(uint8(255), true)
	f.Fuzz(func(t *testing.T, bits uint8, checked bool) {
		nullable := func(value uint8) string {
			switch value % 3 {
			case 1:
				return `,"nullable":false`
			case 2:
				return `,"nullable":true`
			default:
				return ""
			}
		}
		target := nullable(bits & 3)
		member := nullable((bits >> 2) & 3)
		outer := nullable((bits >> 4) & 3)
		effective := member
		if outer != "" {
			effective = outer
		}
		leaf, counts := `{"type":"string"}`, ""
		if checked {
			leaf, counts = `{"type":"string","minLength":2}`, `,"minProperties":1`
		}
		end := fmt.Sprintf(`,"Holder":{"properties":{"value":{"$ref":"#/components/schemas/View"}}},"ZMap":{"additionalProperties":%s%s%s}}`, leaf, target, counts)
		ref := `{"$ref":"#/components/schemas/ZMap"` + member + `}`
		members := []string{ref, `{},` + ref, ref + `,{"description":"note"}`, `{"title":"view"},` + ref + `,{}`}[bits>>6]
		wrapped := schemas(`{"View":{"allOf":[` + members + `]` + outer + `}` + end)
		alias := schemas(`{"View":{"$ref":"#/components/schemas/ZMap"` + effective + `}` + end)
		rq := request(wrapped, "Dto")
		rq.Options.Validation = checked
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("wrapper rejected: %v", err)
		}
		rq.Document = alias
		want, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatalf("control alias rejected: %v", err)
		}
		if !reflect.DeepEqual(got.Artifacts, want.Artifacts) || !reflect.DeepEqual(got.Warnings, want.Warnings) {
			t.Fatalf("wrapper differs from nullable-equivalent alias: bits=%d checked=%t", bits, checked)
		}
	})
}
