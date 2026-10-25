package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

type retainedLockedAdoptionReport struct {
	RetainedOwner        string                    `json:"retained_owner"`
	BeforeVersion        uint32                    `json:"before_version"`
	AfterVersion         uint32                    `json:"after_version"`
	ManifestBeforeSHA256 string                    `json:"manifest_before_sha256"`
	ManifestAfterSHA256  string                    `json:"manifest_after_sha256"`
	BeforeOwners         []string                  `json:"before_owners"`
	AfterOwners          []string                  `json:"after_owners"`
	RetainedFiles        []ownerRenameFile         `json:"retained_files"`
	RetainedLock         lockedAdoptionLock        `json:"retained_lock"`
	Candidates           []lockedReassignmentOwner `json:"candidates"`
	Impact               string                    `json:"impact"`
}

func executeRetainedLockedAdoption(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	invalid := func() (int, report) {
		return 4, failed("E_RETAINED_LOCKED_ADOPTION_INVALID", "Retained adoption requires nonempty locked candidates and distinct explicit retained ownership in one existing root.")
	}
	if len(cfg.Specs) == 0 {
		return invalid()
	}
	names := []string{}
	roots := map[string]bool{}
	for _, spec := range cfg.Specs {
		a := settings[spec.Name]
		if !a.lockHoistedNames || spec.Name == args.retainLockedOwner {
			return invalid()
		}
		roots[a.output] = true
		names = append(names, spec.Name)
	}
	if len(roots) != 1 {
		return invalid()
	}
	root := ""
	for path := range roots {
		root = path
	}
	end := timer.phase("name_lock_read")
	prior, e := output.ReadRetainedLockedAdoptionState(root, args.retainLockedOwner, names)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	source := rootMoveLockState(prior.NameLock)
	for _, spec := range cfg.Specs {
		a := settings[spec.Name]
		if source.Profile != (schema.NameLockProfile{Package: a.packageName, Suffix: a.suffix, Qualified: a.qualifyHoistedNames}) {
			return 1, failed("E_NAME_LOCK_PROFILE", "Every retained-adoption candidate must use the original stored profile.")
		}
	}
	budget, e := output.NewSplitAdoptionBudget(prior.Previous.Files)
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	if e := budget.Add(args.retainLockedOwner, core.GenerationArtifacts{Files: prior.Previous.Files, NameLock: prior.NameLock}); e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	generated := map[string]core.GenerationArtifacts{}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Specs: []specReport{}}
	for _, spec := range cfg.Specs {
		a := settings[spec.Name]
		a.projectNameLock = true
		g, code, diagnostics := readGeneratePrevious(a, timer, prior.NameLock)
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: spec.Name}
		}
		if e := budget.Add(spec.Name, g.Artifacts); e != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}, Spec: spec.Name}
		}
		generated[spec.Name] = g.Artifacts
		entry := specReport{Name: spec.Name, Files: []string{}, Warnings: g.Warnings}
		for _, file := range g.Artifacts.Files {
			entry.Files = append(entry.Files, file.Path)
		}
		all.Specs = append(all.Specs, entry)
		all.Warnings = append(all.Warnings, g.Warnings...)
	}
	timer.plannedConfig(map[string]output.OwnerUpdate{root: {Generated: generated}})
	if denied := configDeniedWarnings(all.Specs, settings, timer); len(denied) != 0 {
		all.Status = "error"
		all.Diagnostics = denied
		return 1, all
	}
	next, e := output.PlanRetainedLockedAdoption(prior, generated)
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	var manifest struct{ Owners map[string][]string }
	_ = json.Unmarshal(next.Manifest, &manifest)
	hash := adoptionSHA256(prior.NameLock)
	migration := retainedLockedAdoptionReport{RetainedOwner: args.retainLockedOwner, BeforeVersion: prior.Previous.Version, AfterVersion: 4, ManifestBeforeSHA256: adoptionSHA256(prior.Previous.Manifest), ManifestAfterSHA256: adoptionSHA256(next.Manifest), BeforeOwners: unionKeys(prior.Previous.Owners, prior.Previous.Owners), AfterOwners: unionKeys(manifest.Owners, manifest.Owners), RetainedFiles: []ownerRenameFile{}, RetainedLock: lockedAdoptionLock{hash, hash, source.Profile, nameLockSymbols(source, prior.Previous.Files)}, Candidates: []lockedReassignmentOwner{}, Impact: "Explicit retained ownership keeps the complete original Kotlin inventory and exact original raw lock. Candidate owners gain only verified unchanged file/allocation subsets; existing peers stay unchanged. Only owned metadata changes, preserving Kotlin bytes/modes/mtimes and copied raw lock attributes under complete pre-input/root identity/attribute CAS and manifest-last normal rollback. Keep the original full input/options under the retained named owner for later updates; direct version3 commands refuse version4. No crash atomicity or automatic recovery."}
	hashes := fileHashes(prior.Previous.Files)
	for _, file := range prior.Previous.Files {
		migration.RetainedFiles = append(migration.RetainedFiles, ownerRenameFile{file.Path, hashes[file.Path]})
	}
	candidateFiles := map[string]bool{}
	for _, name := range prior.OwnerNames {
		raw := next.NameLocks[name]
		state, e := schema.ParseNameLock(raw, source.Profile.Package, source.Profile.Suffix, source.Profile.Qualified)
		if e != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
		}
		entry := lockedReassignmentOwner{Owner: name, BeforeFiles: append([]string{}, prior.Previous.Owners[name]...), AfterFiles: []string{}, AfterSHA256: adoptionSHA256(raw), Symbols: nameLockSymbols(state, generated[name].Files)}
		if old, exists := prior.Previous.NameLocks[name]; exists {
			entry.BeforeSHA256 = adoptionSHA256(old)
		}
		for _, file := range generated[name].Files {
			entry.AfterFiles = append(entry.AfterFiles, file.Path)
			candidateFiles[file.Path] = true
		}
		migration.Candidates = append(migration.Candidates, entry)
	}
	all.Files = unionKeys(candidateFiles, candidateFiles)
	all.RetainedLockedAdoption = &migration
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end = timer.phase(phase)
	changed, e := output.WriteRetainedLockedAdoption(root, output.RetainedLockedAdoptionUpdate{Previous: prior, Generated: generated}, readonly)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	if args.check && changed {
		all.Status = "different"
		return 3, all
	}
	return 0, all
}

func printRetainedLockedAdoption(w io.Writer, r retainedLockedAdoptionReport) {
	_, _ = fmt.Fprintf(w, "retained locked owner %s (manifest v%d -> v%d): SHA256 %s -> %s\n%s\n", r.RetainedOwner, r.BeforeVersion, r.AfterVersion, r.ManifestBeforeSHA256, r.ManifestAfterSHA256, r.Impact)
	_, _ = fmt.Fprintf(w, "  complete retained lock: SHA256=%s; owners %s -> %s\n", r.RetainedLock.AfterSHA256, strconv.Quote(fmt.Sprint(r.BeforeOwners)), strconv.Quote(fmt.Sprint(r.AfterOwners)))
	for _, file := range r.RetainedFiles {
		_, _ = fmt.Fprintf(w, "  retained %s: %s\n", file.Path, file.SHA256)
	}
	for _, symbol := range r.RetainedLock.Symbols {
		_, _ = fmt.Fprintf(w, "  source %s %s descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
	}
	for _, owner := range r.Candidates {
		_, _ = fmt.Fprintf(w, "  candidate %s: files %s -> %s; lock SHA256 %s -> %s\n", owner.Owner, strconv.Quote(fmt.Sprint(owner.BeforeFiles)), strconv.Quote(fmt.Sprint(owner.AfterFiles)), owner.BeforeSHA256, owner.AfterSHA256)
		for _, symbol := range owner.Symbols {
			_, _ = fmt.Fprintf(w, "    %s %s descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
		}
	}
}
