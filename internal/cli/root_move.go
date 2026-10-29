package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"

	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

type rootMoveFile struct {
	Before string   `json:"before"`
	After  string   `json:"after"`
	SHA256 string   `json:"sha256"`
	Owners []string `json:"owners"`
}
type rootMoveLock struct {
	Owner string `json:"owner,omitempty"`
	Path  string `json:"path"`
	ownerRenameLock
}
type rootMoveReport struct {
	Version              uint32         `json:"version"`
	Identity             bool           `json:"identity"`
	ManifestSHA256       string         `json:"manifest_sha256"`
	Files                []rootMoveFile `json:"files"`
	NameLocks            []rootMoveLock `json:"name_locks"`
	Impact               string         `json:"impact"`
	MovedOwners          []string       `json:"moved_owners,omitempty"`
	RetainedOwners       []string       `json:"retained_owners,omitempty"`
	TargetManifestSHA256 string         `json:"target_manifest_sha256,omitempty"`
	TargetOwners         []string       `json:"target_owners,omitempty"`
	SharedTargetFiles    []string       `json:"shared_target_files,omitempty"`
}

func moveCandidate(s output.RootMoveState, generated map[string]core.GenerationArtifacts) *core.Diagnostic {
	invalid := func() *core.Diagnostic {
		return &core.Diagnostic{Code: "E_OUTPUT_ROOT_MOVE_INVALID", Message: "Move requires the complete original owner inventory in one root, without adoption or rename."}
	}
	changed := func() *core.Diagnostic {
		return &core.Diagnostic{Code: "E_OUTPUT_ROOT_MOVE_CHANGED", Message: "Move must preserve exact Kotlin membership and bytes and existing lock mode/profile/allocations/occurrences; update the spec separately."}
	}
	named := s.Version == 2 || s.Version == 4
	if named {
		if len(s.Owners) != len(generated) {
			return invalid()
		}
		for owner := range s.Owners {
			if _, ok := generated[owner]; !ok {
				return invalid()
			}
		}
	} else if len(generated) != 1 {
		return invalid()
	} else if _, ok := generated[""]; !ok {
		return invalid()
	}
	files := map[string]core.GeneratedFile{}
	for _, f := range s.Files {
		files[f.Path] = f
	}
	for _, owner := range unionKeys(generated, generated) {
		a := generated[owner]
		paths := s.Owners[owner]
		if !named {
			for _, f := range s.Files {
				paths = append(paths, f.Path)
			}
		}
		desired := append([]core.GeneratedFile(nil), a.Files...)
		sort.Slice(desired, func(i, j int) bool { return desired[i].Path < desired[j].Path })
		if len(paths) != len(desired) {
			return changed()
		}
		for i, path := range paths {
			if files[path] != desired[i] {
				return changed()
			}
		}
		prior, locked := s.NameLocks[owner]
		if locked != (a.NameLock != nil) {
			return changed()
		}
		if locked {
			// Generation validates the current profile. Semantic comparison permits
			// noncanonical valid old serialization; commit copies its exact old bytes.
			before := rootMoveLockState(prior)
			p := before.Profile
			after, e := schema.ParseNameLock(a.NameLock, p.Package, p.Suffix, p.Qualified)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(before, after) {
				return changed()
			}
		}
	}
	return nil
}

func rootMoveLockState(data []byte) *schema.HoistedNameLock {
	var envelope struct {
		Profile schema.NameLockProfile `json:"profile"`
	}
	_ = json.Unmarshal(data, &envelope)
	state, _ := schema.ParseNameLock(data, envelope.Profile.Package, envelope.Profile.Suffix, envelope.Profile.Qualified)
	return state // ReadRootMoveState has already validated every owned payload.
}

func rootMoveInventory(s output.RootMoveState) rootMoveReport {
	hash := func(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
	r := rootMoveReport{Version: s.Version, ManifestSHA256: hash(s.Manifest), Files: []rootMoveFile{}, NameLocks: []rootMoveLock{}, Impact: "Complete manifest-owned output moves from the explicitly selected source to target with identical Kotlin and metadata bytes, file modes and mtimes, manifest version, owner names and shared membership. Update consumer source directories and keep the target for subsequent generation. Unowned files stay in each root. This point-in-time report is not an approval token, cross-root crash atomicity or automatic recovery guarantee."}
	membership := map[string][]string{}
	for _, owner := range unionKeys(s.Owners, s.Owners) {
		for _, path := range s.Owners[owner] {
			membership[path] = append(membership[path], owner)
		}
	}
	for _, f := range s.Files {
		owners := membership[f.Path]
		if owners == nil {
			owners = []string{}
		}
		r.Files = append(r.Files, rootMoveFile{Before: f.Path, After: f.Path, SHA256: hash([]byte(f.Content)), Owners: owners})
	}
	for _, owner := range unionKeys(s.NameLocks, s.NameLocks) {
		data := s.NameLocks[owner]
		// ReadRootMoveState already performed strict validation. Decode just the
		// validated profile to select that same parser's expected profile.
		state := rootMoveLockState(data)
		path := ".openapi-kt-names.json"
		if owner != "" {
			path = ".openapi-kt-names-" + hash([]byte(owner)) + ".json"
		}
		ownedFiles := s.Files
		if owner != "" {
			ownedFiles = nil
			wanted := map[string]bool{}
			for _, p := range s.Owners[owner] {
				wanted[p] = true
			}
			for _, f := range s.Files {
				if wanted[f.Path] {
					ownedFiles = append(ownedFiles, f)
				}
			}
		}
		r.NameLocks = append(r.NameLocks, rootMoveLock{Owner: owner, Path: path, ownerRenameLock: ownerRenameLock{SHA256: hash(data), Profile: state.Profile, Symbols: nameLockSymbols(state, ownedFiles)}})
	}
	return r
}

func executeDirectRootMove(args arguments, timer *metricCollector) (int, report) {
	end := timer.phase("name_lock_read")
	s, e := output.ReadRootMoveState(args.moveOutputFrom)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	if s.Version != 1 && s.Version != 3 {
		return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Direct root move requires existing version1 or version3 single-owner output.")
	}
	s, e = output.BindRootMoveTarget(s, args.output, "")
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	prior, locked := s.NameLocks[""]
	if locked != args.lockHoistedNames {
		return 4, failed("E_OUTPUT_ROOT_MOVE_CHANGED", "Root move must preserve the existing name-lock mode.")
	}
	generated, code, diagnostics := readGeneratePrevious(args, timer, prior)
	if code != 0 {
		return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics}
	}
	timer.planned(generated.Artifacts)
	return finishRootMove(args, s, map[string]core.GenerationArtifacts{"": generated.Artifacts}, report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Warnings: generated.Warnings}, timer, nil)
}

// rootMoveSelection leaves ordinary --spec parsing and emission override scope
// unchanged. Only the move adapter interprets --move-owner as a set.
func rootMoveSelection(args arguments) []string {
	if args.spec != "" {
		return []string{args.spec}
	}
	owners := append([]string(nil), args.moveOwners...)
	sort.Strings(owners)
	return owners
}

func executeConfigRootMove(args arguments, cfg config.Config, settings map[string]arguments, dir string, timer *metricCollector) (int, report) {
	// Validate every declaration before state/input reads. The explicit pair
	// selects one complete source inventory; other disjoint config roots remain
	// outside its generation/ownership control.
	declaredSettings := map[string]arguments{}
	base := args
	base.explicit = nil
	for _, spec := range cfg.Specs {
		a := configArguments(base, config.Overlay(cfg.Defaults, spec.Settings), dir)
		if a.output == "" {
			return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Every configured spec must declare a nonempty output root before the move target override.")
		}
		declaredSettings[spec.Name] = a
	}
	if len(cfg.Specs) == 0 {
		return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Configuration root move requires a nonempty original inventory.")
	}
	if err := validateRuntimeRoots(declaredSettings); err != nil {
		return 1, failed("E_CONFIG_RUNTIME_ROOTS", "An explicit shared runtime package requires one declared output root across its configured specs.")
	}

	end := timer.phase("name_lock_read")
	s, e := output.ReadRootMoveState(args.moveOutputFrom)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	if s.Version != 2 && s.Version != 4 {
		return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Configured root move requires existing version2 or version4 named-owner output.")
	}
	sourceSpecs := []config.Spec{}
	sourceRoots, otherRoots := map[string]bool{}, map[string]string{}
	for _, spec := range cfg.Specs {
		if _, owned := s.Owners[spec.Name]; owned {
			sourceSpecs = append(sourceSpecs, spec)
			sourceRoots[declaredSettings[spec.Name].output] = true
		} else {
			otherRoots[spec.Name] = declaredSettings[spec.Name].output
		}
	}
	if len(sourceSpecs) != len(s.Owners) || len(sourceRoots) != 1 {
		return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Move requires every original source owner with unchanged names in one declared source or target root.")
	}
	owners := rootMoveSelection(args)
	selected := map[string]bool{}
	for _, owner := range owners {
		if _, ok := s.Owners[owner]; !ok {
			return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Every selected owner must belong to the explicit source inventory.")
		}
		selected[owner] = true
	}
	declared := ""
	for root := range sourceRoots {
		declared = root
	}

	if len(owners) != 0 {
		s, e = output.BindRootMoveOwnerTarget(s, args.output, declared, owners)
	} else {
		s, e = output.BindRootMoveTarget(s, args.output, declared)
	}
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	s, e = output.BindRootMoveConfigPeers(s, otherRoots)
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	// All original declared/CLI spellings have passed storage validation and
	// binding. Normalize only this comparison view; keep the raw declarations
	// and captured bindings unchanged for subsequent CAS checks.
	placement := map[string]arguments{}
	source, sourceErr := filepath.Abs(args.moveOutputFrom)
	target, targetErr := filepath.Abs(args.output)
	if sourceErr != nil || targetErr != nil {
		return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Move requires concrete source and target roots.")
	}
	for name, a := range settings {
		root, err := filepath.Abs(declaredSettings[name].output)
		if err != nil {
			return 4, failed("E_OUTPUT_ROOT_MOVE_INVALID", "Move requires concrete declared roots.")
		}
		a.output = root
		if _, owned := s.Owners[name]; owned {
			a.output = source
			if len(owners) == 0 || selected[name] {
				a.output = target
			}
		}
		placement[name] = a
	}
	if err := validateRuntimeRoots(placement); err != nil {
		return 1, failed("E_CONFIG_RUNTIME_ROOTS", "An explicit shared runtime package requires one output root across its configured specs after the move.")
	}
	generated := map[string]core.GenerationArtifacts{}
	counts := map[string]output.OwnerUpdate{args.output: {Generated: generated}}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Specs: []specReport{}}
	for _, spec := range sourceSpecs {
		a := settings[spec.Name]
		prior, locked := s.NameLocks[spec.Name]
		if locked != a.lockHoistedNames {
			return 4, failed("E_OUTPUT_ROOT_MOVE_CHANGED", "Root move must preserve every owner's name-lock mode.")
		}
		g, code, diagnostics := readGeneratePrevious(a, timer, prior)
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: spec.Name}
		}
		generated[spec.Name] = g.Artifacts
		entry := specReport{Name: spec.Name, Files: []string{}, Warnings: g.Warnings}
		for _, f := range g.Artifacts.Files {
			entry.Files = append(entry.Files, f.Path)
		}
		all.Specs = append(all.Specs, entry)
		all.Warnings = append(all.Warnings, g.Warnings...)
	}
	timer.plannedConfig(counts)
	return finishRootMove(args, s, generated, all, timer, settings)
}

func finishRootMove(args arguments, s output.RootMoveState, generated map[string]core.GenerationArtifacts, all report, timer *metricCollector, settings map[string]arguments) (int, report) {
	var denied core.Diagnostics
	if settings != nil {
		denied = configDeniedWarnings(all.Specs, settings, timer)
	} else if args.denyWarnings {
		end := timer.phase("warning_policy")
		denied = deniedWarnings(all.Warnings, args.allowWarnings)
		end()
	}
	if len(denied) != 0 {
		all.Status = "error"
		all.Diagnostics = denied
		return 1, all
	}

	if e := moveCandidate(s, generated); e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end := timer.phase(phase)
	var changed bool
	var e *core.Diagnostic
	// Both raw roots have already passed output validation/binding. Equivalent
	// absolute clean spellings are the same explicit no-op root; canonical
	// aliases with different absolute paths still reach the storage refusal.
	source := args.moveOutputFrom
	if src, err := filepath.Abs(source); err == nil {
		if dst, err := filepath.Abs(args.output); err == nil && filepath.Clean(src) == filepath.Clean(dst) {
			source = args.output
		}
	}
	owners := rootMoveSelection(args)
	if len(owners) != 0 {
		changed, e = output.MoveOutputOwners(source, args.output, s, owners, readonly)
	} else {
		changed, e = output.MoveOutputRoot(source, args.output, s, readonly)
	}
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	inventory := s
	selected := map[string]bool{}
	if len(owners) != 0 {
		inventory.NameLocks = map[string][]byte{}
		wanted := map[string]bool{}
		for _, owner := range owners {
			selected[owner] = true
			if payload, ok := s.NameLocks[owner]; ok {
				inventory.NameLocks[owner] = payload
			}
			for _, path := range s.Owners[owner] {
				wanted[path] = true
			}
		}
		// Keep full original membership authority for shared file reports, while
		// selecting the distinct moved file/lock union.
		inventory.Files = nil
		for _, file := range s.Files {
			if wanted[file.Path] {
				inventory.Files = append(inventory.Files, file)
			}
		}
	}
	migration := rootMoveInventory(inventory)
	if len(owners) != 0 {
		migration.MovedOwners = owners
		for _, owner := range unionKeys(s.Owners, s.Owners) {
			if !selected[owner] {
				migration.RetainedOwners = append(migration.RetainedOwners, owner)
			}
		}
		migration.Impact = "The selected named owners move together to the explicit target; peers remain at source. Shared payloads are copied to each retaining root. Kotlin and owned name-lock payload bytes, modes and mtimes stay unchanged; source and target manifests retain their version with their respective owner membership. Update configured output roots and consumer source directories. Both roots stage under sorted cooperative locks with ordinary rollback; no cross-root crash atomicity or automatic recovery."
	}
	if s.MergeTarget != nil {
		hash := sha256.Sum256(s.MergeTarget.Manifest)
		migration.TargetManifestSHA256 = hex.EncodeToString(hash[:])
		migration.TargetOwners = unionKeys(s.MergeTarget.Owners, s.MergeTarget.Owners)
		targetPaths := map[string]bool{}
		for _, file := range s.MergeTarget.Files {
			targetPaths[file.Path] = true
		}
		for _, file := range inventory.Files {
			if targetPaths[file.Path] {
				migration.SharedTargetFiles = append(migration.SharedTargetFiles, file.Path)
			}
		}
		sort.Strings(migration.SharedTargetFiles)
		migration.Impact = "Selected source ownership merges into the explicitly observed same-version named target. All existing target owners and Kotlin/raw lock payloads remain unchanged. Newly copied Kotlin/raw locks preserve source bytes, modes and mtimes; retained source peers/shared payloads remain. Only manifest membership is rewritten. Update configured roots and consumer source directories. Complete both-side state is checked under sorted cooperative locks, with all-root staging and normal rollback; no cross-root crash atomicity or automatic recovery."
		if len(migration.SharedTargetFiles) != 0 {
			migration.Impact += " Shared target Kotlin paths retain the target's original bytes, modes and mtimes; source attributes apply only to newly copied payloads."
		}
	}
	migration.Identity = !changed
	all.OutputRootMigration = &migration
	for _, file := range inventory.Files {
		all.Files = append(all.Files, file.Path)
	}
	if args.check && changed {
		all.Status = "different"
		return 3, all
	}
	return 0, all
}

func printRootMove(w io.Writer, r rootMoveReport) {
	_, _ = fmt.Fprintf(w, "output root move (manifest v%d, identity=%t): SHA256=%s\n%s\n", r.Version, r.Identity, r.ManifestSHA256, r.Impact)
	if len(r.MovedOwners) != 0 {
		_, _ = fmt.Fprintf(w, "  moved owners=%s retained owners=%s\n", strconv.Quote(fmt.Sprint(r.MovedOwners)), strconv.Quote(fmt.Sprint(r.RetainedOwners)))
	}
	if r.TargetManifestSHA256 != "" {
		_, _ = fmt.Fprintf(w, "  target manifest: SHA256=%s owners=%s\n", r.TargetManifestSHA256, strconv.Quote(fmt.Sprint(r.TargetOwners)))
	}
	for _, path := range r.SharedTargetFiles {
		_, _ = fmt.Fprintf(w, "  shared target/%s: retain original target bytes/mode/mtime\n", path)
	}
	for _, f := range r.Files {
		_, _ = fmt.Fprintf(w, "  source/%s -> target/%s: %s owners=%s\n", f.Before, f.After, f.SHA256, strconv.Quote(fmt.Sprint(f.Owners)))
	}
	for _, lock := range r.NameLocks {
		_, _ = fmt.Fprintf(w, "  name-lock owner=%s path=%s SHA256=%s package=%s suffix=%s qualified=%t\n", lock.Owner, lock.Path, lock.SHA256, lock.Profile.Package, strconv.Quote(lock.Profile.Suffix), lock.Profile.Qualified)
		for _, symbol := range lock.Symbols {
			_, _ = fmt.Fprintf(w, "  %s %s; descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
		}
	}
}
