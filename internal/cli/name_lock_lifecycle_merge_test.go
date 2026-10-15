package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func lifecycleMergedCore(t testing.TB, input string, locked bool) core.GenerationArtifacts {
	t.Helper()
	a, err := core.Generate(core.GenerationRequest{Document: read(t, input), Format: core.JSON, Options: core.GenerationOptions{
		PackageName: "org.example.dedup", ModelNameSuffix: "Dto", Validation: true,
		HoistSharedShapes: true, QualifyHoistedNames: true, DeduplicateModels: true,
		MergeIdenticalModels: true, LockHoistedNames: locked,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func lifecycleMergedInventory(t testing.TB, state []byte) []lifecycleInventory {
	t.Helper()
	inventory := lifecycleInventories(t, state)
	for i, entry := range inventory {
		switch entry.Role {
		case "ResponseDto":
			inventory[i].Descriptor = "org.example.dedup.models.AnchorDto"
		case "ModeDto":
			inventory[i].Descriptor = "org.example.dedup.models.ModeDto"
		default:
			t.Fatal("unexpected role in small descriptor fixture", entry.Role)
		}
	}
	if len(inventory) != 2 {
		t.Fatal("descriptor fixture must exercise both DTO and open enum", inventory)
	}
	return inventory
}

func lifecycleMergedAliases(t testing.TB, root, owner string) {
	t.Helper()
	for _, target := range []struct{ role, canonical string }{{"Response", "Anchor"}, {"Mode", "Mode"}} {
		name := owner + target.role + "Dto"
		source := string(read(t, filepath.Join(root, "org/example/dedup/models", name+".kt")))
		if !strings.Contains(source, "public typealias "+name+" = org.example.dedup.models."+target.canonical+"Dto") {
			t.Fatal("fixture did not emit shared-to-component descriptor alias", name, source)
		}
		api := string(read(t, filepath.Join(root, "org/example/dedup/api/models", name+".kt")))
		if !strings.Contains(api, "public typealias "+name+" = org.example.dedup.models.") {
			t.Fatal("fixture did not retain API root alias", name, api)
		}
	}
}

func TestNameLockLifecycleNativeMergedCanonicalDescriptors(t *testing.T) {
	for _, action := range []string{"reset", "disable"} {
		t.Run(action, func(t *testing.T) {
			base := t.TempDir()
			input := filepath.Join(base, "merged.json")
			put(t, input, read(t, filepath.Join(repo, "tests/fixtures/name-lock-lifecycle/merge-descriptors.json")))
			root := filepath.Join(base, "generated")
			args := append(lifecycleArgs(input, root, true), "--merge-identical-models")
			result(t, invoke(t, base, args...), 0)
			lifecycleMergedAliases(t, root, "Alpha")
			state := read(t, filepath.Join(root, ".openapi-kt-names.json"))
			before := snapshot(t, root)
			lockGrow(t, input)
			probe := lifecycleMergedCore(t, input, true)
			fresh := lifecycleMergedCore(t, input, action == "reset")
			if !reflect.DeepEqual(probe.Files, fresh.Files) {
				t.Fatal("fresh locked descriptor probe changed unlocked source bytes")
			}
			actionArgs := lifecycleAction(append(lifecycleArgs(input, root, action == "reset"), "--merge-identical-models"), action)
			checkOut := invoke(t, base, append(actionArgs, "--check")...)
			validateOut := invoke(t, base, lifecycleValidate(actionArgs)...)
			if bytes.Contains(checkOut.Stdout, []byte(base)) || bytes.Contains(validateOut.Stdout, []byte(base)) {
				t.Fatal("descriptor migration report leaked an absolute source/output path")
			}
			checked := lifecycleMigrations(t, checkOut, 3)
			validated := lifecycleMigrations(t, validateOut, 0)
			if len(checked) != 1 || !reflect.DeepEqual(checked, validated) || !reflect.DeepEqual(before, snapshot(t, root)) {
				t.Fatal("merged lifecycle readonly behavior/report mismatch")
			}
			lifecycleReportInventories(t, checked[0], action, "", lifecycleMergedInventory(t, state), lifecycleMergedInventory(t, probe.NameLock), before, fresh.Files)
			renamedRoles := map[string]bool{}
			for _, change := range checked[0].Changes {
				if change.Before == nil || change.After == nil || *change.Before == *change.After {
					continue
				}
				if change.DescriptorBefore == nil || change.DescriptorAfter == nil || *change.DescriptorBefore != *change.DescriptorAfter || *change.Before == *change.DescriptorBefore || *change.After == *change.DescriptorAfter || change.ShapeChanged {
					t.Fatal("allocated classifier rename must preserve the merged canonical descriptor", change)
				}
				renamedRoles[change.Role] = true
			}
			if !renamedRoles["ResponseDto"] || !renamedRoles["ModeDto"] {
				t.Fatal("merged report did not cover both actual DTO and enum aliases", checked[0])
			}
			commitOut := invoke(t, t.TempDir(), actionArgs...)
			if bytes.Contains(commitOut.Stdout, []byte(base)) {
				t.Fatal("committed descriptor migration report leaked an absolute path")
			}
			if committed := lifecycleMigrations(t, commitOut, 0); !reflect.DeepEqual(checked, committed) {
				t.Fatal("merged descriptor report changed on commit/cwd")
			}
			lifecycleSources(t, root, fresh.Files)
			lifecycleMergedAliases(t, root, "Aardvark")
			if action == "reset" {
				if !bytes.Equal(probe.NameLock, read(t, filepath.Join(root, ".openapi-kt-names.json"))) {
					t.Fatal("merged reset did not persist fresh pre-merge allocation state")
				}
			} else if _, err := os.Stat(filepath.Join(root, ".openapi-kt-names.json")); !os.IsNotExist(err) {
				t.Fatal("merged disable left owned state", err)
			}
		})
	}
}
