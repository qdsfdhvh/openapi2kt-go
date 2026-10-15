package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHoistWholeEnvelopeDefaultControl(t *testing.T) {
	TestDeduplicateDefaultControl(t)
}
func TestHoistDesiredSuccess(t *testing.T) {
	for _, qualify := range []bool{false, true} {
		root := t.TempDir()
		args := append(deduplicateArgs(root), "--hoist-shared-shapes")
		if qualify {
			args = append(args, "--qualify-hoisted-names")
		}
		result(t, invoke(t, repo, args...), 0)
		for _, tc := range []struct {
			role   string
			owners []string
		}{{"Response", []string{"Alpha", "Beta"}}, {"Input", []string{"Alpha", "Beta"}}, {"Mode", []string{"Alpha", "Beta"}}} {
			target := tc.role + "Dto"
			if qualify {
				target = "Alpha" + target
			}
			shared, e := os.ReadFile(filepath.Join(root, "org/example/dedup/models", target+".kt"))
			if e != nil || strings.Contains(string(shared), "public typealias") {
				t.Fatal("missing real shared declaration", target, e)
			}
			for _, owner := range tc.owners {
				name := owner + tc.role + "Dto"
				b, e := os.ReadFile(filepath.Join(root, "org/example/dedup/api/models", name+".kt"))
				if e != nil || !strings.Contains(string(b), "public typealias "+name+" = org.example.dedup.models."+target) {
					t.Fatal(name, e, string(b))
				}
			}
		}
		b, e := os.ReadFile(filepath.Join(root, "org/example/dedup/api/models/GuardResponseDto.kt"))
		if e != nil || strings.Contains(string(b), "public typealias") {
			t.Fatal("distinct guard hoisted", e)
		}
	}
}
