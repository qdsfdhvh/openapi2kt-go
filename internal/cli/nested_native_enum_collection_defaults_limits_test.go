package cli_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestNestedNativeEnumCollectionDefaultsNativeBoundsNullMembersAndNoDrop(t *testing.T) {
	p := newProject(t)
	good := nestedNativeEnumDocument(nestedNativeEnumField("LL", `{"type":"integer","enum":[1,2]}`, `[[1]]`, ""))
	put(t, p.Input, good)
	result(t, p.generate(t), 0)
	put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
	before := snapshot(t, p.Output)
	for _, tc := range []struct{ field, code, tail string }{
		{nestedNativeEnumField("LM", `{"type":"integer","enum":[1,2],"minimum":2}`, `[{"a/~":1}]`, ""), "E_SCHEMA_INVALID", "/default/0/a~1~0"},
		{nestedNativeEnumField("LL", `{"type":"integer","enum":[1],"minimum":2}`, `[]`, ""), "E_SCHEMA_INVALID", "/items/items/enum"},
		{nestedNativeEnumField("LL", `{"type":"number","enum":[1],"multipleOf":0.5}`, `[]`, ""), "E_SCHEMA_UNSUPPORTED", "/items/items/multipleOf"},
		{nestedNativeEnumField("LL", `{"type":"integer","enum":[1]}`, `[[]]`, `"minItems":1,`), "E_SCHEMA_INVALID", "/default/0"},
		{`{"type":"array","items":{"type":"array","uniqueItems":true,"items":{"type":"integer","enum":[1]}},"default":[]}`, "E_SCHEMA_UNSUPPORTED", "/items/uniqueItems"},
		{nestedNativeEnumField("LM", `{"type":"boolean","nullable":true,"enum":[null,true,null]}`, `[]`, ""), "E_SCHEMA_INVALID", "/items/additionalProperties/enum/2"},
	} {
		put(t, p.Input, nestedNativeEnumDocument(tc.field))
		failure := fails(t, p.generate(t, "--validation"), 1, tc.code)
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value"+tc.tail || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("checked grammar/bounds failure changed output", tc, failure)
		}
	}
	// Null leaf intent does not require a null enum member; an actual null enum
	// member, unlike a nullable leaf, is admitted only in validation mode.
	for _, members := range []string{`[1,2]`, `[null,1,2]`} {
		field := nestedNativeEnumField("LM", `{"type":["integer","null"],"enum":`+members+`}`, `[{"nil":null,"a":1}]`, "")
		put(t, p.Input, nestedNativeEnumDocument(field))
		result(t, p.generate(t, "--validation"), 0)
		if members == `[1,2]` {
			result(t, p.generate(t), 0)
		} else {
			before := snapshot(t, p.Output)
			failure := result(t, p.generate(t), 1)
			if failure.Status != "error" || len(failure.Files) != 0 || len(failure.Warnings) != 0 || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("null vocabulary admitted unchecked or changed output", failure)
			}
		}
	}
}

func TestNestedNativeEnumCollectionDefaultsNativeSourceBudgets(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		good := nestedNativeEnumDocument(nestedNativeEnumField("LL", `{"type":"integer","enum":[0,1]}`, `[[1]]`, ""))
		put(t, p.Input, good)
		args := nestedNativeEnumArgs(p.Input, p.Output, checked, core.DataClass)
		result(t, invoke(t, p.Root, args...), 0)
		put(t, filepath.Join(p.Output, "Notes.kt"), []byte("handwritten"))
		probe := func(name string, doc []byte, code string) {
			t.Helper()
			t.Run(fmt.Sprintf("%s/checked=%t", name, checked), func(t *testing.T) {
				before := snapshot(t, p.Output)
				put(t, p.Input, doc)
				r, err := core.GenerateWithDiagnostics(core.GenerationRequest{Document: doc, Options: core.GenerationOptions{PackageName: "org.example.catalog", Validation: checked, ModelStyle: core.DataClass}})
				if code == "" {
					if err != nil {
						t.Fatal("core boundary acceptance", err)
					}
					result(t, invoke(t, p.Root, args...), 0)
					for _, f := range r.Artifacts.Files {
						if string(read(t, filepath.Join(p.Output, f.Path))) != f.Content {
							t.Fatal("boundary source parity", f.Path)
						}
					}
					return
				}
				if err == nil || !reflect.DeepEqual(r, core.GenerationReport{}) {
					t.Fatal("core budget returned partial success", err)
				}
				d, ok := err.(core.Diagnostics)
				if !ok || len(d) != 1 || d[0].Code != code {
					t.Fatal("core budget diagnostic", err)
				}
				failure := fails(t, invoke(t, p.Root, args...), 1, code)
				if !reflect.DeepEqual(d[0].Pointer, failure.Diagnostics[0].Pointer) || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("budget original pointer/preservation", failure)
				}
			})
		}
		for _, count := range []int{240, 241} {
			members := make([]string, count)
			for i := range members {
				members[i] = fmt.Sprint(i)
			}
			code := ""
			if count == 241 {
				code = "E_MODEL_LIMIT"
			}
			probe(fmt.Sprintf("raw-enum%d", count), nestedNativeEnumDocument(nestedNativeEnumField("LL", `{"type":"integer","enum":[`+strings.Join(members, ",")+`]}`, `[]`, "")), code)
		}
		// Field depth includes the terminal: 47 collection edges + one leaf.
		// Checked membership itself counts through every edge even without counts.
		depths := []int{47, 48}
		if checked {
			depths = []int{16, 17}
		}
		for i, depth := range depths {
			code := ""
			if i == 1 {
				code = "E_SCHEMA_DEPTH"
			}
			probe(fmt.Sprintf("depth%d", depth), nestedNativeEnumDocument(nestedNativeEnumField(strings.Repeat("L", depth), `{"type":"integer","enum":[0]}`, `[]`, "")), code)
		}
		// Zero tokens fit both caps at exactly 16,384 whole-tree values;
		// the next value crosses the work and compact-byte caps.
		for _, nodes := range []int{16382, 16383} {
			code := ""
			if nodes == 16383 {
				code = "E_MODEL_LIMIT"
			}
			probe(fmt.Sprintf("values%d", nodes+2), nestedNativeEnumDocument(nestedNativeEnumField("LL", `{"type":"integer","enum":[0]}`, `[[`+strings.TrimSuffix(strings.Repeat("0,", nodes), ",")+`]]`, "")), code)
		}
		// Dictionary key bytes count as data, independently of scalar membership.
		for _, size := range []int{32756, 32757} {
			code := ""
			if size == 32757 {
				code = "E_MODEL_LIMIT"
			}
			probe(fmt.Sprintf("default-bytes%d", size+12), nestedNativeEnumDocument(nestedNativeEnumField("MM", `{"type":"integer","enum":[0]}`, `{"`+strings.Repeat("a", size)+`":{"x":0}}`, "")), code)
		}
		for _, count := range []int{240, 241} {
			fields := make([]string, count)
			for i := range fields {
				fields[i] = fmt.Sprintf(`"v%d":%s`, i, nestedNativeEnumField("LL", `{"type":"integer","enum":[0]}`, `[[0]]`, ""))
			}
			code := ""
			if count == 241 {
				code = "E_MODEL_LIMIT"
			}
			probe(fmt.Sprintf("slots%d", count), []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{`+strings.Join(fields, ",")+`}}}}}`), code)
		}
		probe("input8MiB", append(append([]byte(nil), good...), bytes.Repeat([]byte(" "), core.MaxDocumentBytes-len(good))...), "")
		// CLI limits IO before core JSON decoding; use a valid prefix plus spaces.
		before := snapshot(t, p.Output)
		put(t, p.Input, append(append([]byte(nil), good...), bytes.Repeat([]byte(" "), core.MaxDocumentBytes+1-len(good))...))
		fails(t, invoke(t, p.Root, args...), 1, "E_INPUT_LIMIT")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("input cap failure changed output")
		}
	}
}
