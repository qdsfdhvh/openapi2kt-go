package cli_test

import (
	"bytes"
	"reflect"
	"testing"
)

func TestStringEnumCollectionBoundsNativeFrozenCheckIdentity(t *testing.T) {
	shape := `{"properties":{"codes":{"type":"array","items":{"enum":["one","two","x"],"minLength":2,"maxLength":6,"pattern":"^[a-z]+$"},"default":["one"]}}}`
	op := `{"get":{"responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + op + `,"/b":` + op + `}}`)
	p := newProject(t)
	put(t, p.Input, doc)
	flags := []string{"--validation", "--hoist-shared-shapes", "--lock-hoisted-names", "--deduplicate-models", "--merge-identical-models"}
	result(t, p.generate(t, flags...), 0)
	before := snapshot(t, p.Output)
	for _, change := range []struct{ old, new string }{{`"minLength":2`, `"minLength":3`}, {`"maxLength":6`, `"maxLength":5`}, {`"pattern":"^[a-z]+$"`, `"pattern":"[a-z]"`}, {`"default":["one"]`, `"default":["two"]`}} {
		put(t, p.Input, bytes.ReplaceAll(doc, []byte(change.old), []byte(change.new)))
		fails(t, p.generate(t, flags...), 1, "E_NAME_LOCK_CHANGED")
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("Frozen check drift changed output", change)
		}
	}
	put(t, p.Input, doc)
	result(t, p.generate(t, append(flags, "--check")...), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("Replay changed output")
	}
}
