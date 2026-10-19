package core_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"openapi-kt/core"
)

func mergeRequest(t testing.TB) core.GenerationRequest {
	t.Helper()
	input, err := os.ReadFile("../tests/fixtures/http/merge-models.json")
	if err != nil {
		t.Fatal(err)
	}
	r := request(input, "Dto")
	r.Options.PackageName = "org.example.merge"
	r.Options.Validation = true
	return r
}
func TestMergeResolvedRootsAndOpaqueOwners(t *testing.T) {
	r := mergeRequest(t)
	before, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("nondeterministic", err)
	}
	if !reflect.DeepEqual(before.Warnings, got.Warnings) {
		t.Fatal("warning drift")
	}
	for name, target := range map[string]string{"ZetaDto": "AlphaDto", "ChildZDto": "ChildADto", "ParentZDto": "ParentADto", "RecursiveZDto": "RecursiveADto"} {
		if code := source(t, got, "models/"+name); !strings.Contains(code, "public typealias "+name+" = org.example.merge.models."+target) {
			t.Fatal(name, code)
		}
	}
	for _, name := range []string{"NestedADto", "NestedZDto", "DifferentDto", "RecursiveADto"} {
		if code := source(t, got, "models/"+name); !strings.Contains(code, "public data class "+name) {
			t.Fatal("protected/distinct/canonical", name, code)
		}
	}
	if code := source(t, got, "models/ParentADto"); !strings.Contains(code, "org.example.merge.models.ChildADto") {
		t.Fatal(code)
	}
	if code := source(t, got, "api/models/OneResponseDto"); !strings.Contains(code, "= org.example.merge.models.AlphaDto") {
		t.Fatal("component priority", code)
	}
	if code := source(t, got, "api/models/ApplyInputDto"); !strings.Contains(code, "Patch<kotlin.String>") {
		t.Fatal("context loss", code)
	}
}
func TestMergeCatalogSpellingOrder(t *testing.T) {
	r := mergeRequest(t)
	r.Options.MergeIdenticalModels = true
	r.Options.HoistSharedShapes = true
	r.Options.LockHoistedNames = true
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(r.Document, &doc); err != nil {
		t.Fatal(err)
	}
	var components map[string]json.RawMessage
	if err := json.Unmarshal(doc["components"], &components); err != nil {
		t.Fatal(err)
	}
	var catalog map[string]json.RawMessage
	if err := json.Unmarshal(components["schemas"], &catalog); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for name := range catalog {
		names = append(names, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	var spelling bytes.Buffer
	spelling.WriteByte('{')
	for i, name := range names {
		if i > 0 {
			spelling.WriteByte(',')
		}
		key, _ := json.Marshal(name)
		spelling.Write(key)
		spelling.WriteByte(':')
		spelling.Write(catalog[name])
	}
	spelling.WriteByte('}')
	components["schemas"] = json.RawMessage(spelling.Bytes())
	encoded, _ := json.Marshal(components)
	doc["components"] = encoded
	before, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Document, err = json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(before, got) {
		t.Fatal("catalog spelling changed models/client/state/warnings", err)
	}
}

func TestMergeExclusionsNeitherDirection(t *testing.T) {
	for _, excluded := range []string{"AlphaDto", "ZetaDto", "ChildADto", "ChildZDto"} {
		r := mergeRequest(t)
		r.Options.MergeIdenticalModels = true
		r.Options.ExcludeMergeModels = []string{"org.example.merge.models." + excluded}
		got, err := core.GenerateWithDiagnostics(r)
		if err != nil {
			t.Fatal(err)
		}
		if code := source(t, got, "models/"+excluded); !strings.Contains(code, "public data class "+excluded) {
			t.Fatal(excluded, code)
		}
		if excluded == "ChildADto" || excluded == "ChildZDto" {
			if code := source(t, got, "models/ParentZDto"); !strings.Contains(code, "public data class ParentZDto") {
				t.Fatal("distinct excluded child identity erased", code)
			}
		}
	}
}
func TestMergeOptionsBeforeInput(t *testing.T) {
	for _, exclusions := range [][]string{{"Alpha"}, {"org.example.*"}, {"org.example.Alpha()"}, {""}} {
		for _, enabled := range []bool{false, true} {
			r := request([]byte("broken"), "")
			r.Options.MergeIdenticalModels = enabled
			r.Options.ExcludeMergeModels = exclusions
			result, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_OPTIONS" || !reflect.DeepEqual(result, core.GenerationReport{}) {
				t.Fatal(exclusions, err, result)
			}
		}
	}
}
func TestMergeHoistDedupPlainLockProfiles(t *testing.T) {
	for _, plain := range []bool{false, true} {
		for _, qualify := range []bool{false, true} {
			r := mergeRequest(t)
			r.Options.MergeIdenticalModels = true
			r.Options.HoistSharedShapes = true
			r.Options.DeduplicateModels = true
			r.Options.LockHoistedNames = true
			r.Options.QualifyHoistedNames = qualify
			if plain {
				r.Options.ModelStyle = core.PlainClass
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err != nil {
				t.Fatal(err)
			}
			r.PreviousNameLock = got.Artifacts.NameLock
			again, err := core.GenerateWithDiagnostics(r)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatal("lock replay", err)
			}
			found := false
			for _, f := range got.Artifacts.Files {
				if strings.Contains(f.Path, "/api/models/OneResponseDto.kt") {
					found = strings.Contains(f.Content, "typealias OneResponseDto = org.example.merge.models.")
				}
			}
			if !found {
				t.Fatal("old public name lost")
			}
			if plain && strings.Contains(source(t, got, "models/AlphaDto"), "data class") {
				t.Fatal("merge after plain")
			}
		}
	}
}
