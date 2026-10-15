package cli_test

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

// These fixtures have separate Kotlin consumers. Root move proves unchanged
// native source bytes and owned classifier/descriptor inventories; it does not
// claim a fresh JVM runtime result or revalidate unrelated emitter behavior.
func TestRootMoveNativeAcceptedConsumerByteControls(t *testing.T) {
	type profile struct {
		fixture, pkg string
		locked       bool
	}
	cases := []profile{
		{"tests/fixtures/name-lock-lifecycle/kotlin-before.json", "org.example.namelock", true},
		{"tests/fixtures/name-lock-lifecycle/merge-descriptors.json", "org.example.namelock", true},
		{"tests/fixtures/http/servers.json", "org.example.servers", false},
		{"tests/fixtures/http/referenced-collection-field-defaults.json", "org.example.referenceddefaults", false},
	}
	for _, prefix := range []string{"org", "frame", "input", "requestBuilder", "value", "client", "basePath", "json", "httpClient", "shadows"} {
		cases = append(cases, profile{"tests/fixtures/http/namespace-collisions.json", prefix + ".example.namespaces", false})
	}
	for _, style := range []string{"data_class", "plain_class"} {
		for _, checked := range []bool{false, true} {
			for _, p := range cases {
				t.Run(style+"/"+p.pkg+"/"+p.fixture+"/"+map[bool]string{false: "off", true: "on"}[checked], func(t *testing.T) {
					base := t.TempDir()
					src, dst, control := filepath.Join(base, "old"), filepath.Join(base, "moved"), filepath.Join(base, "control")
					args := []string{"generate", "--input", filepath.Join(repo, p.fixture), "--output", src, "--package-name", p.pkg, "--model-name-suffix=Dto", "--model-style", style, "--diagnostic-format=json"}
					if checked {
						args = append(args, "--validation")
					}
					if p.locked {
						args = append(args, "--hoist-shared-shapes", "--qualify-hoisted-names", "--lock-hoisted-names", "--deduplicate-models", "--merge-identical-models")
					}
					result(t, invoke(t, base, args...), 0)
					before := snapshot(t, src)
					run := append([]string(nil), args...)
					run[4] = dst
					run = append(run, "--move-output-from", src)
					migrated := invoke(t, base, run...)
					result(t, migrated, 0)
					after := snapshot(t, dst)
					for path, old := range before {
						if path != ".openapi-kt.lock" && !reflect.DeepEqual(old, after[path]) {
							t.Fatal("consumer source changed", path)
						}
					}
					// Independent fresh generation of exactly these options is the control.
					fresh := append([]string(nil), args...)
					fresh[4] = control
					result(t, invoke(t, base, fresh...), 0)
					for path, f := range snapshot(t, control) {
						if filepath.Ext(path) == ".kt" && f.Bytes != after[path].Bytes {
							t.Fatal("fresh control differs", path)
						}
					}
					if p.locked {
						var r struct {
							Move struct {
								Locks []struct{ Symbols []lifecycleInventory } `json:"name_locks"`
							} `json:"output_root_migration"`
						}
						if e := json.Unmarshal(migrated.Stdout, &r); e != nil {
							t.Fatal(e)
						}
						if len(r.Move.Locks) != 1 {
							t.Fatal("missing descriptor evidence")
						}
						var raw struct {
							Entries []struct {
								Role, Name, Shape string
								Occurrences       []string
							}
						}
						if e := json.Unmarshal([]byte(before[".openapi-kt-names.json"].Bytes), &raw); e != nil {
							t.Fatal(e)
						}
						if len(raw.Entries) != len(r.Move.Locks[0].Symbols) {
							t.Fatal("hoisted inventory lost")
						}
						for _, symbol := range r.Move.Locks[0].Symbols {
							if symbol.Classifier == "" || symbol.Descriptor == "" || symbol.Shape == "" || len(symbol.Occurrences) == 0 {
								t.Fatal("incomplete public descriptor report")
							}
						}
					}
					if bytes.Contains(migrated.Stdout, []byte(base)) {
						t.Fatal("machine path leaked")
					}
				})
			}
		}
	}
}
