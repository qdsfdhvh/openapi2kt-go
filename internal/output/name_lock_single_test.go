package output

import (
	"bytes"
	"encoding/json"
	"openapi-kt/core"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func singleLockRequest(t testing.TB, style core.ModelStyle, lock bool) core.GenerationRequest {
	t.Helper()
	input, e := os.ReadFile("../../tests/fixtures/http/deduplicate-models.json")
	if e != nil {
		t.Fatal(e)
	}
	return core.GenerationRequest{Document: input, Format: core.JSON, Options: core.GenerationOptions{PackageName: "org.example.singlelock", ModelNameSuffix: "Dto", Validation: true, ModelStyle: style, HoistSharedShapes: true, LockHoistedNames: lock}}
}
func singleGenerated(t testing.TB, req core.GenerationRequest) core.GenerationArtifacts {
	t.Helper()
	a, e := core.Generate(req)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestSingleNameLockLegacyEnvelopeControl(t *testing.T) {
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		root := t.TempDir()
		a := singleGenerated(t, singleLockRequest(t, style, false))
		if changed, e := Write(root, a, false); e != nil || !changed {
			t.Fatal(e)
		}
		var m struct {
			Version int
			Files   map[string]string
		}
		data := mustRead(t, filepath.Join(root, manifestName))
		if e := json.Unmarshal(data, &m); e != nil || m.Version != 1 || len(m.Files) != len(a.Files) {
			t.Fatal(e)
		}
		for _, f := range a.Files {
			if !bytes.Equal(mustRead(t, filepath.Join(root, filepath.FromSlash(f.Path))), []byte(f.Content)) || m.Files[f.Path] != digest([]byte(f.Content)) {
				t.Fatal("default envelope lost")
			}
		}
		if _, e := os.Stat(filepath.Join(root, ".openapi-kt-names.json")); !os.IsNotExist(e) {
			t.Fatal("default created metadata", e)
		}
		before := snapshot(t, root)
		for _, check := range []bool{true, false} {
			if changed, e := Write(root, a, check); e != nil || changed {
				t.Fatal(e)
			}
			if !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("default no-op/check mutated tree")
			}
		}
	}
}
func TestSingleNameLockDesiredSuccess(t *testing.T) {
	root := t.TempDir()
	req := singleLockRequest(t, core.DataClass, true)
	a := singleGenerated(t, req)
	if changed, e := WriteNameLocked(root, a, nil, false); e != nil || !changed {
		t.Fatal("desired managed name-lock transaction success", e)
	}
	if !bytes.Equal(a.NameLock, mustRead(t, filepath.Join(root, ".openapi-kt-names.json"))) {
		t.Fatal("state not committed")
	}
	data := mustRead(t, filepath.Join(root, manifestName))
	var m map[string]json.RawMessage
	if e := json.Unmarshal(data, &m); e != nil || len(m) != 3 || string(m["version"]) != "3" {
		t.Fatal("managed manifest", e, string(data))
	}
}
