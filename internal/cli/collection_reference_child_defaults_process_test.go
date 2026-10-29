package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectionReferenceChildDefaultNativePreservation(t *testing.T) {
	for _, checked := range []bool{false, true} {
		p := newProject(t)
		doc := []byte(`{"openapi":"3.1.0","components":{"schemas":{"Word":{"enum":["one","two"]},"Rows":{"additionalProperties":{"$ref":"#/components/schemas/Word","default":"one"},"default":{"a":"two"}},"Holder":{"properties":{"value":{"$ref":"#/components/schemas/Rows"}}}}}}`)
		flags := []string{}
		if checked {
			flags = append(flags, "--validation")
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, flags...), 0)
		put(t, filepath.Join(p.Output, "unowned.txt"), []byte("keep"))
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(flags, "--check")...), 0)
		put(t, p.Input, bytes.Replace(doc, []byte(`"default":"one"`), []byte(`"default":"future"`), 1))
		for _, check := range []bool{false, true} {
			args := append([]string{}, flags...)
			if check {
				args = append(args, "--check")
			}
			failure := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
			if failure.Diagnostics[0].Pointer == nil || *failure.Diagnostics[0].Pointer != "/components/schemas/Rows/additionalProperties/default" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
				t.Fatal("invalid child default changed existing output or location", failure)
			}
		}
		put(t, p.Input, doc)
		result(t, p.generate(t, append(flags, "--check")...), 0)
	}
}
