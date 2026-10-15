package cli_test

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func intrinsicRawProject(t testing.TB, nullable bool) project {
	t.Helper()
	p := newProject(t)
	b := read(t, filepath.Join(repo, "tests/fixtures/http/intrinsic-nullable-raw.json"))
	if !nullable {
		var doc, components, catalog map[string]json.RawMessage
		_ = json.Unmarshal(b, &doc)
		_ = json.Unmarshal(doc["components"], &components)
		_ = json.Unmarshal(components["schemas"], &catalog)
		catalog["ZRaw"] = json.RawMessage(`{"type":"object"}`)
		catalog["ZEmpty"] = json.RawMessage(`{"type":"object","properties":{}}`)
		components["schemas"], _ = json.Marshal(catalog)
		doc["components"], _ = json.Marshal(components)
		b, _ = json.Marshal(doc)
	}
	put(t, p.Input, b)
	return p
}
func TestIntrinsicNullableRawNativeWholeControl(t *testing.T) {
	p := intrinsicRawProject(t, false)
	result(t, p.generate(t), 0)
	result(t, p.generate(t, "--validation"), 0)
}
func TestIntrinsicNullableRawNativeDesiredSuccess(t *testing.T) {
	p := intrinsicRawProject(t, true)
	for _, validation := range []bool{false, true} {
		args := []string{"--model-name-suffix", "Dto"}
		if validation {
			args = append(args, "--validation")
		}
		result(t, p.generate(t, args...), 0)
		before := snapshot(t, p.Output)
		result(t, p.generate(t, append(args, "--check")...), 0)
		result(t, p.generate(t, args...), 0)
		if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("check/noop changed raw files")
		}
	}
	put(t, filepath.Join(p.Output, "handwritten.txt"), []byte("synthetic unowned file"))
	before := snapshot(t, p.Output)
	p.change(t, func(doc map[string]any) { schemas(doc)["ZRaw"].(map[string]any)["required"] = true })
	for _, args := range [][]string{{"--model-name-suffix=Dto"}, {"--model-name-suffix=Dto", "--validation", "--check"}} {
		r := fails(t, p.generate(t, args...), 1, "E_SCHEMA_INVALID")
		if *r.Diagnostics[0].Pointer != "/components/schemas/ZRaw" || !reflect.DeepEqual(before, snapshot(t, p.Output)) {
			t.Fatal("failed input changed output or source", r)
		}
	}
}
