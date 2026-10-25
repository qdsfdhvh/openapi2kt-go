package cli_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRootMoveDisjointConfigRetainsUnrelatedRoots(t *testing.T) {
	for _, selected := range []bool{false, true} {
		for _, locked := range []bool{false, true} {
			base := t.TempDir()
			names := []string{"alpha", "beta", "gamma"}
			lifecycleConfigInputs(t, base, names)
			cfg := lifecycleConfig(t, base, names, locked, false, false)
			text := strings.Replace(string(read(t, cfg)), "name=\"gamma\"", "name=\"gamma\"\noutput_dir='unrelated'", 1)
			put(t, cfg, []byte(text))
			args := []string{"generate", "--config", cfg, "--diagnostic-format=json"}
			result(t, invoke(t, base, args...), 0)
			src, dst, peer := filepath.Join(base, "generated"), filepath.Join(base, "new"), filepath.Join(base, "unrelated")
			sourceBefore, peerBefore := snapshot(t, src), snapshot(t, peer)
			// Unrelated inputs are not original source control for this concrete pair.
			originalUnrelatedInput := read(t, filepath.Join(base, "gamma.json"))
			if err := os.Remove(filepath.Join(base, "gamma.json")); err != nil {
				t.Fatal(err)
			}
			run := append(append([]string(nil), args...), "--move-output-from", src, "--output", dst)
			if selected {
				run = append(run, "--spec=alpha")
			}
			result(t, invoke(t, base, append(append([]string(nil), run...), "--check")...), 3)
			result(t, invoke(t, base, lifecycleValidate(run)...), 0)
			if !reflect.DeepEqual(sourceBefore, snapshot(t, src)) || !reflect.DeepEqual(peerBefore, snapshot(t, peer)) {
				t.Fatal("disjoint check changed old roots")
			}
			result(t, invoke(t, base, run...), 0)
			if !reflect.DeepEqual(peerBefore, snapshot(t, peer)) {
				t.Fatal("move touched unrelated inventory")
			}
			moved := snapshot(t, dst)
			for path, old := range sourceBefore {
				if selected {
					if strings.HasPrefix(path, "org/example/lockalpha/") && !reflect.DeepEqual(old, moved[path]) {
						t.Fatal("selected disjoint payload changed", path)
					}
				} else if path != ".openapi-kt.lock" && !reflect.DeepEqual(old, moved[path]) {
					t.Fatal("complete disjoint-source move changed bytes", path)
				}
			}
			next := text
			if selected {
				next = strings.Replace(next, "name=\"alpha\"", "name=\"alpha\"\noutput_dir='new'", 1)
			} else {
				next = strings.Replace(next, "output_dir='generated'", "output_dir='new'", 1)
			}
			put(t, cfg, []byte(next))
			put(t, filepath.Join(base, "gamma.json"), originalUnrelatedInput)
			afterSource, afterTarget := snapshot(t, src), snapshot(t, dst)
			result(t, invoke(t, base, append(append([]string(nil), args...), "--check")...), 0)
			result(t, invoke(t, base, args...), 0)
			if !reflect.DeepEqual(afterSource, snapshot(t, src)) || !reflect.DeepEqual(afterTarget, snapshot(t, dst)) || !reflect.DeepEqual(peerBefore, snapshot(t, peer)) {
				t.Fatal("ordinary generation changed disjoint move output")
			}

		}
	}
}
