package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

func ownerRenameIdentifier(name string) bool {
	if len(name) == 0 || !(name[0] >= 'a' && name[0] <= 'z' || name[0] >= 'A' && name[0] <= 'Z') {
		return false
	}
	for _, c := range []byte(name) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

type ownerRenameFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type ownerRenameLock struct {
	SHA256  string                 `json:"sha256"`
	Profile schema.NameLockProfile `json:"profile"`
	Symbols []nameLockSymbol       `json:"symbols"`
}
type ownerRenameReport struct {
	OldOwner string            `json:"old_owner"`
	NewOwner string            `json:"new_owner"`
	Version  uint32            `json:"version"`
	Files    []ownerRenameFile `json:"files"`
	NameLock *ownerRenameLock  `json:"name_lock,omitempty"`
	Impact   string            `json:"impact"`
}

func executeOwnerRename(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	selected := map[string]bool{}
	for _, s := range cfg.Specs {
		if args.spec == "" || args.spec == s.Name {
			selected[s.Name] = true
		}
	}
	if len(selected) != len(args.ownerRenames) {
		return 4, failed("E_OWNER_RENAME_INVALID", "Map every selected configured spec with --rename-owner OLD:NEW; NEW must be the selected spec name.")
	}
	for next := range args.ownerRenames {
		if !selected[next] {
			return 4, failed("E_OWNER_RENAME_INVALID", "Rename mappings must name selected configured specs.")
		}
	}
	sourcesByRoot := map[string]map[string]bool{}
	for _, s := range cfg.Specs {
		if selected[s.Name] {
			root := settings[s.Name].output
			if sourcesByRoot[root] == nil {
				sourcesByRoot[root] = map[string]bool{}
			}
			sourcesByRoot[root][args.ownerRenames[s.Name]] = true
		}
	}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Specs: []specReport{}}
	updates := map[string]output.OwnerRenameUpdate{}
	counts := map[string]output.OwnerUpdate{}
	paths := map[string]string{}
	// Read only roots named by the selected effective config. All source generation
	// and warning policy finish before output acquires any locks.
	for _, s := range cfg.Specs {
		if !selected[s.Name] {
			continue
		}
		a := settings[s.Name]
		old := args.ownerRenames[s.Name]
		u, exists := updates[a.output]
		if !exists {
			end := timer.phase("name_lock_read")
			state, e := output.ReadOwnerRenameState(a.output)
			end()
			if e != nil {
				return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}, Spec: s.Name}
			}
			u = output.OwnerRenameUpdate{Previous: state, Renames: map[string]string{}, Generated: map[string]core.GenerationArtifacts{}}
		}
		files, exists := u.Previous.Owners[old]
		if !exists {
			return 4, failed("E_OWNER_RENAME_INVALID", "The old owner must exist in the selected spec's output root.")
		}
		if _, occupied := u.Previous.Owners[s.Name]; occupied && old != s.Name && !sourcesByRoot[a.output][s.Name] {
			return 4, failed("E_OWNER_RENAME_INVALID", "An occupied new owner must also be a selected source in the same output root.")
		}
		prior, locked := u.Previous.NameLocks[old]
		if locked != a.lockHoistedNames {
			return 4, failed("E_OWNER_RENAME_CHANGED", "Owner rename must preserve the existing name-lock mode.")
		}
		generated, code, diagnostics := readGeneratePrevious(a, timer, prior)
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: s.Name}
		}
		migration := ownerRenameReport{OldOwner: old, NewOwner: s.Name, Version: u.Previous.Version, Files: []ownerRenameFile{}, Impact: "Only named ownership and its dedicated metadata filename change. Kotlin membership, source bytes and public names are preserved; unselected peer owners and shared runtime references are retained. Keep the new spec name for subsequent generation. This report is a point-in-time comparison, not an approval token or crash recovery guarantee."}
		hashes := fileHashes(files)
		for _, file := range files {
			migration.Files = append(migration.Files, ownerRenameFile{file.Path, hashes[file.Path]})
			paths[a.output+"\x00"+file.Path] = file.Path
		}
		if locked {
			state, e := schema.ParseNameLock(prior, a.packageName, a.suffix, a.qualifyHoistedNames)
			if e != nil {
				return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}, Spec: s.Name}
			}
			hash := sha256.Sum256(prior)
			migration.NameLock = &ownerRenameLock{SHA256: hex.EncodeToString(hash[:]), Profile: state.Profile, Symbols: nameLockSymbols(state, files)}
		}
		all.OwnerRenames = append(all.OwnerRenames, migration)
		entry := specReport{Name: s.Name, Files: []string{}, Warnings: generated.Warnings}
		for _, file := range generated.Artifacts.Files {
			entry.Files = append(entry.Files, file.Path)
		}
		all.Specs = append(all.Specs, entry)
		all.Warnings = append(all.Warnings, generated.Warnings...)
		u.Renames[s.Name] = old
		u.Generated[s.Name] = generated.Artifacts
		updates[a.output] = u
		count := counts[a.output]
		if count.Generated == nil {
			count.Generated = map[string]core.GenerationArtifacts{}
		}
		count.Generated[s.Name] = generated.Artifacts
		counts[a.output] = count
	}
	timer.plannedConfig(counts)
	if denied := configDeniedWarnings(all.Specs, settings, timer); len(denied) != 0 {
		all.Status = "error"
		all.Diagnostics = denied
		all.OwnerRenames = nil
		return 1, all
	}

	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end := timer.phase(phase)
	changed, e := output.WriteOwnerRenamesBatch(updates, readonly)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	for _, path := range paths {
		all.Files = append(all.Files, path)
	}
	sort.Strings(all.Files)
	sort.Slice(all.OwnerRenames, func(i, j int) bool { return all.OwnerRenames[i].NewOwner < all.OwnerRenames[j].NewOwner })
	if args.check && changed {
		all.Status = "different"
		return 3, all
	}
	return 0, all
}
