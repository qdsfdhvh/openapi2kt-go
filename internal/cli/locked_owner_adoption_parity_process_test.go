package cli_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These are the shared actual consumer fixtures used by the existing Kotlin
// feature suites. Fresh native controls bind all files to the current emitter;
// this test is byte/API preservation evidence, not fresh JVM execution.
func TestLockedOwnerAdoptionNativeCurrentConsumerByteControls(t *testing.T) {
	cases := []struct{ fixture, pkg, client, consumer string }{}
	for _, prefix := range []string{"org", "frame", "input", "requestBuilder", "value", "client", "basePath", "json", "httpClient", "shadows"} {
		cases = append(cases, struct{ fixture, pkg, client, consumer string }{"namespace-collisions", prefix + ".example.namespaces", "", "NamespaceCollisionContracts.kt"})
	}
	for _, c := range []struct{ fixture, pkg, client, consumer string }{
		{"servers", "org.example.servers", "", "ServerContracts.kt"},
		{"string-enum-collection-defaults", "org.example.stringenumcollections", "StringEnumCollections", "StringEnumCollectionDefaultsContracts.kt"},
		{"nested-builtin-collection-defaults", "org.example.nesteddefaults", "NestedDefaults", "NestedBuiltinCollectionDefaultsContracts.kt"},
		{"referenced-collection-field-defaults", "org.example.referenceddefaults", "ReferencedDefaults", "ReferencedCollectionFieldDefaultsContracts.kt"},
		{"deduplicate-models", "org.example.dedup", "", ""},
	} {
		cases = append(cases, c)
	}
	for _, c := range cases {
		for _, style := range []string{"data_class", "plain_class"} {
			for _, checked := range []bool{false, true} {
				if c.fixture == "deduplicate-models" && !checked {
					continue // This fixture's original bounds require validation.
				}
				t.Run(c.fixture+"/"+c.pkg+"/"+style+fmt.Sprint(checked), func(t *testing.T) {
					base := t.TempDir()
					input := filepath.Join(base, "input.json")
					put(t, input, read(t, filepath.Join(repo, "tests/fixtures/http", c.fixture+".json")))
					root := filepath.Join(base, "generated")
					direct := []string{"generate", "--input", input, "--output", root, "--package-name=" + c.pkg, "--model-name-suffix=Dto", "--model-style=" + style, "--hoist-shared-shapes", "--lock-hoisted-names", "--diagnostic-format=json"}
					if checked {
						direct = append(direct, "--validation")
					}
					if c.client != "" {
						direct = append(direct, "--api-client-name="+c.client)
					}
					result(t, invoke(t, base, direct...), 0)
					before := snapshot(t, root)
					// Original consumer fixtures use the same declarations. Compare their
					// ordinary generation too; do not edit consumer guards for adoption.
					control := filepath.Join(base, "control")
					controlArgs := []string{}
					for _, arg := range direct {
						if arg == "--lock-hoisted-names" {
							continue
						}
						if arg == root {
							arg = control
						}
						controlArgs = append(controlArgs, arg)
					}
					result(t, invoke(t, base, controlArgs...), 0)
					controls := snapshot(t, control)
					for path, entry := range before {
						if strings.HasSuffix(path, ".kt") && entry.Bytes != controls[path].Bytes {
							t.Fatal("fresh unlocked/locked source differs", path)
						}
					}
					var consumer []byte
					consumerPath := filepath.Join(repo, "samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts", c.consumer)
					if c.consumer != "" {
						consumer = read(t, consumerPath)
					}
					cfg := filepath.Join(base, "adopt.toml")
					put(t, cfg, []byte(fmt.Sprintf("config_version=+1\n[\"defaults\"]\noutput_dir='generated'\npackage_name=%q\nmodel_name_suffix='Dto'\nmodel_style=%q\napi_client_name=%q\nhoist_shared_shapes=true\nlock_hoisted_names=true\nvalidation=%t\n[[\"specs\"]]\nname='catalog'\ninput=\"\"\"input.json\"\"\"\n", c.pkg, style, c.client, checked)))
					run := []string{"generate", "--config", cfg, "--spec=catalog", "--adopt-locked-owner", "--diagnostic-format=json"}
					result(t, invoke(t, base, append(run, "--check")...), 3)
					result(t, invoke(t, base, lifecycleValidate(run)...), 0)
					if !reflect.DeepEqual(before, snapshot(t, root)) {
						t.Fatal("consumer readonly differs")
					}
					result(t, invoke(t, base, run...), 0)
					after := snapshot(t, root)
					for path, entry := range before {
						if strings.HasSuffix(path, ".kt") && !reflect.DeepEqual(entry, after[path]) {
							t.Fatal("actual API byte/mtime/mode changed", path)
						}
					}
					if before[".openapi-kt-names.json"].Bytes != after[namedLockPath("catalog")].Bytes {
						t.Fatal("consumer raw lock changed")
					}
					result(t, invoke(t, base, "generate", "--config", cfg, "--spec=catalog", "--check", "--diagnostic-format=json"), 0)
					if c.consumer != "" && !reflect.DeepEqual(consumer, read(t, consumerPath)) {
						t.Fatal("actual consumer changed")
					}
				})
			}
		}
	}
}
