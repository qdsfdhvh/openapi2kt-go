package core_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedNativeEnumCollectionDefaultsIndependentSourceCheckedAndSlotCaps(t *testing.T) {
	// These are independent probes, not a Cartesian JVM width/depth promise.
	for _, edges := range []int{47, 48} {
		r := request(fieldDoc(nestedDefaultChain(edges, `{"type":"integer","enum":[0]}`, `[]`)), "Dto")
		if edges == 47 {
			if _, err := core.GenerateWithDiagnostics(r); err != nil {
				t.Fatal("source48", err)
			}
		} else {
			nestedNativeEnumDefaultFailure(t, r, "E_SCHEMA_DEPTH", "/components/schemas/Record/properties/value"+strings.Repeat("/items", edges))
		}
	}
	for _, edges := range []int{16, 17} {
		r := request(fieldDoc(nestedDefaultChain(edges, `{"type":"integer","enum":[0]}`, `[]`)), "Dto")
		r.Options.Validation = true // Membership itself requires recursive checks.
		got, err := core.GenerateWithDiagnostics(r)
		if edges == 16 && err != nil || edges == 17 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("checked16", edges, err)
		}
	}
	for _, checked := range []bool{false, true} {
		for _, count := range []int{240, 241} {
			fields := make([]string, count)
			for i := range fields {
				fields[i] = fmt.Sprintf(`"v%d":%s`, i, nestedDefaultChain(2, `{"type":"integer","format":"int64","enum":[0]}`, `[[0]]`))
			}
			r := request(schemas(`{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}`), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if count == 240 && err != nil || count == 241 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal("slots240", count, checked, err)
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsRawEnumBudgetCountsNull(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, withNull := range []bool{false, true} {
			if withNull && !checked {
				continue // Explicit null native vocabulary is validation-only.
			}
			for _, count := range []int{240, 241} {
				members := make([]string, count)
				for i := range members {
					members[i] = fmt.Sprint(i)
				}
				if withNull {
					members[0] = "null"
				}
				field := nestedDefaultChain(2, `{"type":["integer","null"],"enum":[`+strings.Join(members, ",")+`]}`, `[]`)
				r := request(fieldDoc(field), "Dto")
				r.Options.Validation = checked
				if count == 240 {
					if _, err := core.GenerateWithDiagnostics(r); err != nil {
						t.Fatal("raw240", checked, withNull, err)
					}
				} else {
					nestedNativeEnumDefaultFailure(t, r, "E_MODEL_LIMIT", "/components/schemas/Record/properties/value/items/items/enum")
				}
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsOriginalCanonicalBytesAndWholeTreeNodes(t *testing.T) {
	for _, checked := range []bool{false, true} {
		for _, size := range []int{32760, 32761} {
			annotation := `{"` + strings.Repeat("k", size) + `":[0]}`
			if len(annotation) != size+8 {
				t.Fatal("bad independent byte probe")
			}
			field := `{"additionalProperties":{"type":"array","items":{"type":"integer","enum":[0]}},"default":` + annotation + `}`
			r := request(fieldDoc(field), "Dto")
			r.Options.Validation = checked
			got, err := core.GenerateWithDiagnostics(r)
			if size == 32760 && err != nil || size == 32761 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
				t.Fatal("original32KiB", size, checked, err)
			}
		}
		for _, count := range []int{16382, 16383} {
			annotation := `[[` + strings.TrimSuffix(strings.Repeat("0,", count), ",") + `]]`
			r := request(fieldDoc(nestedDefaultChain(2, `{"type":"integer","enum":[0]}`, annotation)), "Dto")
			r.Options.Validation = checked
			if count == 16382 {
				// 16,384 nodes including both containers; 32,767 raw bytes.
				if len(annotation) != 32767 {
					t.Fatal("bad independent node probe")
				}
				if _, err := core.GenerateWithDiagnostics(r); err != nil {
					t.Fatal("tree16384", checked, err)
				}
			} else {
				// Work preflight closes at the original final node, before its
				// bytes or membership could be considered/canonicalized.
				nestedNativeEnumDefaultFailure(t, r, "E_MODEL_LIMIT", "/components/schemas/Record/properties/value/default/0/16382")
			}
		}
		// Source token 1e-9 is four bytes; canonical Double 1e-09 is five.
		// The original 30,003 bytes fit, but the canonical 36,003 do not.
		annotation := `[[` + strings.TrimSuffix(strings.Repeat("1e-9,", 6000), ",") + `]]`
		if len(annotation) != 30003 {
			t.Fatal("bad canonical byte probe")
		}
		r := request(fieldDoc(nestedDefaultChain(2, `{"type":"number","format":"double","enum":[1e-9]}`, annotation)), "Dto")
		r.Options.Validation = checked
		nestedNativeEnumDefaultFailure(t, r, "E_MODEL_LIMIT", "/components/schemas/Record/properties/value/default")
	}
}

func FuzzNestedNativeEnumCollectionDefaultsBoundedOriginalTrees(f *testing.F) {
	for _, value := range []string{`[]`, `[[1,null],null,[]]`, `[[1.000000001]]`, `[[1.00000000000000001]]`, `[{"a/~":false}]`, `{"$ref":{"type":1}}`, `{"z":{"z":3,"a":2},"a":4}`} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 2048 || !json.Valid([]byte(value)) {
			return
		}
		for _, prefix := range []string{
			`{"type":"array","items":{"type":"array","nullable":true,"items":{"type":"integer","nullable":true,"enum":[1,2]}}`,
			`{"type":"array","items":{"additionalProperties":{"type":"boolean","enum":[true]}}`,
			`{"additionalProperties":{"additionalProperties":{"type":"number","format":"float","nullable":true,"enum":[1,2]}}`,
		} {
			for _, checked := range []bool{false, true} {
				r := nestedNativeEnumDefaultRequest(prefix+`,"default":`+value+`}`, checked, core.DataClass)
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					if _, ok := err.(core.Diagnostics); !ok || !reflect.DeepEqual(got, core.GenerationReport{}) {
						t.Fatal("unstructured or partial failure", err)
					}
					continue
				}
				again, err := core.GenerateWithDiagnostics(r)
				if err != nil || !reflect.DeepEqual(got, again) {
					t.Fatal("nondeterministic original native annotation", err)
				}
			}
		}
	})
}
