package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInlineBuiltinCollectionChildDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"array","items":{"type":"string","default":"element"},"default":[]}}}}}}`)
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
		put(t, p.Input, bytes.Replace(doc, []byte(`"default":"element"`), []byte(`"default":true`), 1))
		for _, check := range []bool{false, true} {
			args := append([]string{}, flags...)
			if check {
				args = append(args, "--check")
			}
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value/items/default" ||
				!reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid child default changed existing output or location", failure)
			}
		}
		failure := fails(t, invoke(t, p.Root, validate...), 1, "E_SCHEMA_INVALID")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Holder/properties/value/items/default" ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("invalid child validation changed output or location", failure)
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}

func TestInlineBuiltinCollectionChildDefaultHierarchyNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Root":{"oneOf":[{"$ref":"#/components/schemas/Leaf"}],"discriminator":{"propertyName":"kind","mapping":{"leaf":"Leaf"}}},"Leaf":{"properties":{"kind":{"type":"string","const":"leaf"},"rows":{"type":"array","items":{"type":"string"}}}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		annotated := bytes.Replace(doc, []byte(`"items":{"type":"string"}`),
			[]byte(`"items":{"type":"string","default":"x"}`), 1)
		put(t, p.Input, annotated)
		failure := fails(t, p.generate(t, flags...), 1, "E_SCHEMA_UNSUPPORTED")
		if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Leaf/properties/rows/items/default" ||
			!reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("hierarchy child annotation changed existing output or location", failure)
		}
	}
}

func TestInlineBuiltinCollectionChildOwnerDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"value":{"type":"array","items":{"type":"integer","default":7}}},"default":{"value":[2]}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, flags...), 0)
		before := snapshot(t, p.Output)
		for _, tc := range []struct {
			doc     []byte
			pointer string
		}{
			{bytes.Replace(doc, []byte(`"default":7`), []byte(`"default":"bad"`), 1), "/components/schemas/Holder/properties/value/items/default"},
			{bytes.Replace(doc, []byte(`"value":[2]`), []byte(`"value":["bad"]`), 1), "/components/schemas/Holder/default/value/0"},
		} {
			put(t, p.Input, tc.doc)
			for _, check := range []bool{false, true} {
				args := append([]string{}, flags...)
				if check {
					args = append(args, "--check")
				}
				failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
				if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != tc.pointer ||
					!reflect.DeepEqual(before, snapshot(t, p.Output)) {
					t.Fatal("invalid independent default changed existing output or location", failure)
				}
			}
		}
	}
}

func TestInlineSpecialCollectionChildDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Holder":{"properties":{"ids":{"type":"array","items":{"type":"string","format":"uuid","default":"123E4567E89B12D3A456426614174000"}},"dates":{"type":"object","additionalProperties":{"type":"string","format":"date","default":"2024-02-29"}},"times":{"type":"array","items":{"type":"string","format":"date-time","default":"2024-02-29T12:34:56Z"}}},"default":{}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, doc)
		result(t, invoke(t, p.Root, append([]string{"validate", "--input", p.Input, "--diagnostic-format=json"}, flags...)...), 0)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		for _, tc := range []struct {
			valid, invalid, pointer string
		}{
			{`"123E4567E89B12D3A456426614174000"`, `"not-a-uuid"`, "/components/schemas/Holder/properties/ids/items/default"},
			{`"2024-02-29"`, `"2023-02-29"`, "/components/schemas/Holder/properties/dates/additionalProperties/default"},
			{`"2024-02-29T12:34:56Z"`, `"2024-02-29T12:34:56.1234567890Z"`, "/components/schemas/Holder/properties/times/items/default"},
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
					t.Fatal("invalid special child changed existing output or location", checked, check, failure)
				}
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(append([]string{}, flags...), "--check")...), 0)
	}
}
