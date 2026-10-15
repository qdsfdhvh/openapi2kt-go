package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestRecoveryRejectionNativeConfigRetainsJournalAndBackups(t *testing.T) {
	for _, locked := range []bool{false, true} {
		t.Run(strconv.FormatBool(locked), func(t *testing.T) {
			dir, cfg := configFiles(t)
			if locked {
				put(t, cfg, []byte("config_version=1\n[defaults]\ninput="+strconv.Quote(filepath.Join(repo, "tests/fixtures/http/deduplicate-models.json"))+"\noutput_dir='generated'\npackage_name='org.example.recovery'\nmodel_name_suffix='Dto'\nvalidation=true\nhoist_shared_shapes=true\nlock_hoisted_names=true\n[[specs]]\nname='alpha'\n"))
			}
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, repo, args...), 0)
			root := filepath.Join(dir, "generated")
			stage := filepath.Join(root, ".openapi-kt-stage-interrupted")
			if err := os.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			put(t, filepath.Join(stage, "transaction.json"), []byte(`{"version":1,"operations":[{"path":"kept.kt","backup":"old-0","staged":"new-0","had_old":true}]}`))
			put(t, filepath.Join(stage, "old-0"), []byte("synthetic old bytes"))
			put(t, filepath.Join(stage, "new-0"), []byte("synthetic new bytes"))
			before := snapshot(t, root)
			for _, extra := range [][]string{nil, {"--check"}, {"--spec", "alpha"}, {"--spec", "alpha", "--check"}, {"--reconcile"}} {
				invocation := append(append([]string{}, args...), extra...)
				fails(t, invoke(t, repo, invocation...), 4, "E_RECOVERY_REQUIRED")
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("config recovery refusal changed backups/output", locked, extra)
				}
			}
			if locked {
				fails(t, invoke(t, repo, "validate", "--config", cfg, "--spec", "alpha", "--diagnostic-format=json"), 4, "E_RECOVERY_REQUIRED")
				if !reflect.DeepEqual(before, snapshot(t, root)) {
					t.Fatal("locked validation changed recovery evidence")
				}
			}
		})
	}
}

func TestRecoveryRejectionNativeLockedDirectRetainsEvidence(t *testing.T) {
	root := t.TempDir()
	args := append(deduplicateArgs(root), "--hoist-shared-shapes", "--lock-hoisted-names")
	result(t, invoke(t, repo, args...), 0)
	// deduplicateArgs writes under root/generated.
	output := filepath.Join(root, "generated")
	for i, arg := range args {
		if arg == "--output" {
			output = args[i+1]
		}
	}
	if err := os.Mkdir(filepath.Join(output, ".openapi-kt-stage-interrupted"), 0700); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(output, ".openapi-kt-stage-interrupted", "transaction.json"), []byte("malformed retained journal must not be consumed"))
	put(t, filepath.Join(output, ".openapi-kt-stage-interrupted", "old-0"), []byte("synthetic retained backup"))
	before := snapshot(t, output)
	for _, check := range []bool{false, true} {
		invocation := append([]string{}, args...)
		if check {
			invocation = append(invocation, "--check")
		}
		fails(t, invoke(t, repo, invocation...), 4, "E_RECOVERY_REQUIRED")
		if !reflect.DeepEqual(before, snapshot(t, output)) {
			t.Fatal("locked direct recovery mutated evidence")
		}
	}
}
