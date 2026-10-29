package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestScalarReferenceChildDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Number":{"type":"integer"},"Date":{"type":"string","format":"date"},"Bytes":{"type":"string","format":"byte"},"Holder":{"properties":{"numbers":{"type":"array","items":{"$ref":"#/components/schemas/Number","default":3}},"dates":{"additionalProperties":{"$ref":"#/components/schemas/Date","default":"2024-02-29"}},"bytes":{"type":"array","items":{"$ref":"#/components/schemas/Bytes","default":[-128,127]}}}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, validate...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		for _, tc := range []struct{ valid, invalid, pointer string }{
			{`"default":3`, `"default":true`, "/components/schemas/Holder/properties/numbers/items/default"},
			{`"default":"2024-02-29"`, `"default":"2023-02-29"`, "/components/schemas/Holder/properties/dates/additionalProperties/default"},
			{`"default":[-128,127]`, `"default":[-129,127]`, "/components/schemas/Holder/properties/bytes/items/default/0"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.valid), []byte(tc.invalid), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid reference child changed output or location", checked, check, failure)
				}
			}
			failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid validation changed output or location", checked, failure)
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}

func TestScalarReferenceChildIndependentFactoryNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Number":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Number","default":2},"Holder":{"properties":{"values":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":3}}}}}}}`)
	put(t, p.Input, doc)
	flags := []string{"--validation"}
	validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
	result(t, invoke(t, p.Root, validate...), 0)
	result(t, p.generate(t, flags...), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ valid, invalid, pointer string }{
		{`"default":1`, `"default":true`, "/components/schemas/Number/default"},
		{`"default":2`, `"default":true`, "/components/schemas/Alias/default"},
		{`"default":3`, `"default":true`, "/components/schemas/Holder/properties/values/items/default"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.valid), []byte(tc.invalid), 1))
		for _, args := range [][]string{
			{"--validation"},
			{"--validation", "--check"},
		} {
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid independent annotation changed output or location", failure)
			}
		}
		failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid independent annotation changed output during validation", failure)
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestScalarReferenceOuterDefaultsNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Number":{"type":"integer","default":1},"Alias":{"$ref":"#/components/schemas/Number","default":2},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":3},"default":[4]},"RowsAlias":{"$ref":"#/components/schemas/Rows","default":[5]},"Holder":{"properties":{"values":{"$ref":"#/components/schemas/RowsAlias","default":[6]}},"default":{"values":[7]}}}}}`)
	put(t, p.Input, doc)
	flags := []string{"--validation"}
	validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
	result(t, invoke(t, p.Root, validate...), 0)
	result(t, p.generate(t, flags...), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ valid, invalid, pointer string }{
		{`"default":1`, `"default":true`, "/components/schemas/Number/default"},
		{`"default":2`, `"default":true`, "/components/schemas/Alias/default"},
		{`"default":3`, `"default":true`, "/components/schemas/Rows/items/default"},
		{`"default":[4]`, `"default":[true]`, "/components/schemas/Rows/default/0"},
		{`"default":[5]`, `"default":[true]`, "/components/schemas/RowsAlias/default/0"},
		{`"default":[6]`, `"default":[true]`, "/components/schemas/Holder/properties/values/default/0"},
		{`"default":{"values":[7]}`, `"default":{"values":[true]}`, "/components/schemas/Holder/default/values/0"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.valid), []byte(tc.invalid), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid outer annotation changed output or location", failure)
			}
		}
		failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid outer annotation changed output during validation", failure)
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestScalarReferenceNestedFactoriesNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Number":{"type":"integer","default":1},"Values":{"additionalProperties":{"$ref":"#/components/schemas/Number","default":2},"default":{"inner":3}},"ValuesAlias":{"$ref":"#/components/schemas/Values","default":{"alias":4}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias"},"default":[{"row":5}]},"Holder":{"properties":{"values":{"$ref":"#/components/schemas/Rows","default":[{"field":6}]}},"default":{"values":[{"owner":7}]}}}}}`)
	put(t, p.Input, doc)
	result(t, invoke(t, p.Root, "validate", "--input", p.Input, "--diagnostic-format=json", "--validation"), 0)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, tc := range []struct{ valid, invalid, pointer string }{
		{`"default":2`, `"default":true`, "/components/schemas/Values/additionalProperties/default"},
		{`"default":{"inner":3}`, `"default":{"inner":true}`, "/components/schemas/Values/default/inner"},
		{`"default":{"alias":4}`, `"default":{"alias":true}`, "/components/schemas/ValuesAlias/default/alias"},
		{`"default":[{"row":5}]`, `"default":[{"row":true}]`, "/components/schemas/Rows/default/0/row"},
		{`"default":[{"field":6}]`, `"default":[{"field":true}]`, "/components/schemas/Holder/properties/values/default/0/field"},
		{`"default":{"values":[{"owner":7}]}`, `"default":{"values":[{"owner":true}]}`, "/components/schemas/Holder/default/values/0/owner"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(tc.valid), []byte(tc.invalid), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid nested factory changed output or location", failure)
			}
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestReferencedCollectionChildNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Leaf":{"type":"integer"},"Values":{"additionalProperties":{"$ref":"#/components/schemas/Leaf"}},"Alias":{"$ref":"#/components/schemas/Values"},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":{"child":2}},"default":[{"root":3}]}}}}`)
	put(t, p.Input, doc)
	validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
	result(t, invoke(t, p.Root, validate...), 0)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, bad := range []struct{ replacement, pointer string }{
		{`"default":{"child":true}`, "/components/schemas/Rows/items/default/child"},
		{`"default":false`, "/components/schemas/Rows/items/default"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(`"default":{"child":2}`), []byte(bad.replacement), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid collection child changed output or location", failure)
			}
		}
		failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid collection child changed output during validation", failure)
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestReferencedEnumCollectionChildNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Word":{"enum":["one","two","three","four","five"],"default":"one"},"WordAlias":{"$ref":"#/components/schemas/Word","default":"two"},"Values":{"type":"array","items":{"$ref":"#/components/schemas/WordAlias","default":"three"},"default":["four"]},"ValuesAlias":{"$ref":"#/components/schemas/Values","default":["five"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/ValuesAlias","default":["one"]}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, validate...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		for _, tc := range []struct{ original, replacement, pointer string }{
			{`"default":"one"`, `"default":"future"`, "/components/schemas/Word/default"},
			{`"default":"two"`, `"default":"future"`, "/components/schemas/WordAlias/default"},
			{`"default":"three"`, `"default":"future"`, "/components/schemas/Values/items/default"},
			{`"default":["four"]`, `"default":["future"]`, "/components/schemas/Values/default/0"},
			{`"default":["five"]`, `"default":["future"]`, "/components/schemas/ValuesAlias/default/0"},
			{`"default":["one"]`, `"default":["future"]`, "/components/schemas/Rows/items/default/0"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.original), []byte(tc.replacement), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid enum collection annotation changed output or location", checked, check, failure)
				}
			}
			failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid enum collection annotation changed output during validation", checked, failure)
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}

func TestReferencedEnumIntermediateCollectionChildNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Word":{"enum":["one","two"]},"Inner":{"type":"array","items":{"$ref":"#/components/schemas/Word"},"default":["one"]},"InnerAlias":{"$ref":"#/components/schemas/Inner","default":["two"]},"Values":{"additionalProperties":{"$ref":"#/components/schemas/InnerAlias","default":["one"]},"default":{"root":["two"]}},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Values","default":{"child":["one"]}}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		validate := append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, validate...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
		for _, tc := range []struct{ original, replacement, pointer string }{
			{`"default":{"child":["one"]}`, `"default":{"child":["future"]}`, "/components/schemas/Rows/items/default/child/0"},
			{`"default":{"root":["two"]}`, `"default":{"root":["future"]}`, "/components/schemas/Values/default/root/0"},
		} {
			put(t, p.Input, bytes.Replace(doc, []byte(tc.original), []byte(tc.replacement), 1))
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid intermediate collection annotation changed output or location", checked, check, failure)
				}
			}
			failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid intermediate collection annotation changed output during validation", checked, failure)
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}

func TestInlineScalarCollectionChildNativePreservation(t *testing.T) {
	p := newProject(t)
	doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Values":{"type":"array","items":{"type":"integer"},"default":[1]},"Alias":{"$ref":"#/components/schemas/Values","default":[2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":[3]}}}}}`)
	put(t, p.Input, doc)
	validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
	result(t, invoke(t, p.Root, validate...), 0)
	result(t, p.generate(t, "--validation"), 0)
	put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	for _, bad := range []struct{ original, replacement, pointer string }{
		{`"default":[1]`, `"default":[false]`, "/components/schemas/Values/default/0"},
		{`"default":[2]`, `"default":[false]`, "/components/schemas/Alias/default/0"},
		{`"default":[3]`, `"default":[false]`, "/components/schemas/Rows/items/default/0"},
		{`"default":[3]`, `"default":false`, "/components/schemas/Rows/items/default"},
	} {
		put(t, p.Input, bytes.Replace(doc, []byte(bad.original), []byte(bad.replacement), 1))
		for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid inline-scalar collection child changed output or location", failure)
			}
		}
		failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid inline-scalar child changed output during validation", failure)
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, "--validation", "--check"), 0)
}

func TestInlineSpecialCollectionChildNativePreservation(t *testing.T) {
	for _, tc := range []struct {
		name, doc string
		bad       []struct{ original, replacement, pointer string }
	}{
		{
			name: "date",
			doc:  `{"openapi":"3.1.0","components":{"schemas":{"Values":{"type":"array","items":{"type":"string","format":"date"},"default":["2024-02-29"]},"Alias":{"$ref":"#/components/schemas/Values","default":["2024-03-01"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":["2024-04-01"]}}}}}`,
			bad: []struct{ original, replacement, pointer string }{
				{`"default":["2024-02-29"]`, `"default":["2023-02-29"]`, "/components/schemas/Values/default/0"},
				{`"default":["2024-03-01"]`, `"default":["2023-02-29"]`, "/components/schemas/Alias/default/0"},
				{`"default":["2024-04-01"]`, `"default":["2023-02-29"]`, "/components/schemas/Rows/items/default/0"},
			},
		},
		{
			name: "byte",
			doc:  `{"openapi":"3.1.0","components":{"schemas":{"Values":{"type":"array","items":{"type":"string","format":"byte"},"default":[[1]]},"Alias":{"$ref":"#/components/schemas/Values","default":[[2]]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":[[3]]}}}}}`,
			bad: []struct{ original, replacement, pointer string }{
				{`"default":[[1]]`, `"default":[[128]]`, "/components/schemas/Values/default/0/0"},
				{`"default":[[2]]`, `"default":[[128]]`, "/components/schemas/Alias/default/0/0"},
				{`"default":[[3]]`, `"default":[[128]]`, "/components/schemas/Rows/items/default/0/0"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProject(t)
			doc := []byte(tc.doc)
			put(t, p.Input, doc)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
			result(t, invoke(t, p.Root, validate...), 0)
			result(t, p.generate(t, "--validation"), 0)
			put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
			before := snapshot(t, p.Output)
			result(t, p.generate(t, "--validation", "--check"), 0)
			for _, bad := range tc.bad {
				put(t, p.Input, bytes.Replace(doc, []byte(bad.original), []byte(bad.replacement), 1))
				for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
					failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
						!reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("invalid special collection annotation changed output or location", failure)
					}
				}
				failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid special collection annotation changed output during validation", failure)
				}
			}
			put(t, p.Input, doc)
			result(t, p.generate(t, "--validation", "--check"), 0)
		})
	}
}

func TestInlineEnumCollectionChildNativePreservation(t *testing.T) {
	for _, tc := range []struct {
		name, doc string
		bad       []struct{ original, replacement, pointer string }
	}{
		{
			name: "string",
			doc:  `{"openapi":"3.1.0","components":{"schemas":{"Values":{"type":"array","items":{"type":"string","enum":["one","two"]},"default":["one"]},"Alias":{"$ref":"#/components/schemas/Values","default":["two"]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":["one"]}}}}}`,
			bad: []struct{ original, replacement, pointer string }{
				{`"default":["one"]`, `"default":["future"]`, "/components/schemas/Values/default/0"},
				{`"default":["two"]`, `"default":["future"]`, "/components/schemas/Alias/default/0"},
				{`"default":["one"]}}`, `"default":["future"]}}`, "/components/schemas/Rows/items/default/0"},
			},
		},
		{
			name: "integer",
			doc:  `{"openapi":"3.1.0","components":{"schemas":{"Values":{"type":"array","items":{"type":"integer","enum":[1,2]},"default":[1]},"Alias":{"$ref":"#/components/schemas/Values","default":[2]},"Rows":{"type":"array","items":{"$ref":"#/components/schemas/Alias","default":[1]}}}}}`,
			bad: []struct{ original, replacement, pointer string }{
				{`"default":[1]`, `"default":[3]`, "/components/schemas/Values/default/0"},
				{`"default":[2]`, `"default":[3]`, "/components/schemas/Alias/default/0"},
				{`"default":[1]}}`, `"default":[3]}}`, "/components/schemas/Rows/items/default/0"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newProject(t)
			doc := []byte(tc.doc)
			put(t, p.Input, doc)
			validate := []string{"validate", "--input", p.Input, "--diagnostic-format=json", "--validation"}
			result(t, invoke(t, p.Root, validate...), 0)
			result(t, p.generate(t, "--validation"), 0)
			put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
			before := snapshot(t, p.Output)
			result(t, p.generate(t, "--validation", "--check"), 0)
			for _, bad := range tc.bad {
				put(t, p.Input, bytes.Replace(doc, []byte(bad.original), []byte(bad.replacement), 1))
				for _, args := range [][]string{{"--validation"}, {"--validation", "--check"}} {
					failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
					if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
						!reflect.DeepEqual(before, snapshot(t, p.Output)) {
						t.Fatal("invalid enum collection annotation changed output or location", failure)
					}
				}
				failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != bad.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid enum collection annotation changed output during validation", failure)
				}
			}
			put(t, p.Input, doc)
			result(t, p.generate(t, "--validation", "--check"), 0)
		})
	}
}
