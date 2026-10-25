package cli

import (
	"fmt"
	"io"
	"strconv"

	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

type lockedReassignmentOwner struct {
	Owner        string           `json:"owner"`
	BeforeFiles  []string         `json:"before_files"`
	AfterFiles   []string         `json:"after_files"`
	BeforeSHA256 string           `json:"before_lock_sha256,omitempty"`
	AfterSHA256  string           `json:"after_lock_sha256,omitempty"`
	Symbols      []nameLockSymbol `json:"symbols"`
}
type lockedOwnerReassignmentReport struct {
	Version              uint32                    `json:"version"`
	ManifestBeforeSHA256 string                    `json:"manifest_before_sha256"`
	ManifestAfterSHA256  string                    `json:"manifest_after_sha256"`
	BeforeOwners         []string                  `json:"before_owners"`
	AfterOwners          []string                  `json:"after_owners"`
	Files                []ownerRenameFile         `json:"files"`
	Owners               []lockedReassignmentOwner `json:"owners"`
	SourceSymbols        []nameLockSymbol          `json:"source_symbols"`
	Impact               string                    `json:"impact"`
}

func executeLockedOwnerReassignment(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	invalid := func() (int, report) {
		return 4, failed("E_LOCKED_OWNER_REASSIGN_INVALID", "Locked owner reassignment requires a complete nonempty desired locked inventory in one existing root.")
	}
	if len(cfg.Specs) == 0 {
		return invalid()
	}
	roots := map[string]bool{}
	names := []string{}
	for _, spec := range cfg.Specs {
		a := settings[spec.Name]
		if !a.lockHoistedNames {
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
	prior, e := output.ReadLockedOwnerReassignmentState(root, names)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	budget, e := output.NewSplitAdoptionBudget(prior.Previous.Files)
	if e != nil {
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
	next, e := output.PlanLockedOwnerReassignment(prior, generated)
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	source := rootMoveLockState(prior.NameLock)
	migration := lockedOwnerReassignmentReport{Version: 4, ManifestBeforeSHA256: adoptionSHA256(prior.Previous.Manifest), ManifestAfterSHA256: adoptionSHA256(next.Manifest), BeforeOwners: unionKeys(prior.Previous.Owners, prior.Previous.Owners), AfterOwners: prior.OwnerNames, Files: []ownerRenameFile{}, Owners: []lockedReassignmentOwner{}, SourceSymbols: nameLockSymbols(source, prior.Previous.Files), Impact: "Only same-root owner memberships and owned lock metadata change. Complete original Kotlin bytes, modes and mtimes, historical allocation names and original occurrence union remain. Unchanged semantic lock payloads retain original raw bytes; removed owners lose only their owned metadata. Use the desired full configuration afterward. Complete pre-input original state, root identity and attributes participate in locked CAS and manifest-last normal rollback; no crash atomicity or automatic recovery."}
	hashes := fileHashes(prior.Previous.Files)
	for _, file := range prior.Previous.Files {
		all.Files = append(all.Files, file.Path)
		migration.Files = append(migration.Files, ownerRenameFile{file.Path, hashes[file.Path]})
	}
	ownerSet := map[string]bool{}
	for name := range prior.Previous.Owners {
		ownerSet[name] = true
	}
	for name := range generated {
		ownerSet[name] = true
	}
	for _, name := range unionKeys(ownerSet, ownerSet) {
		entry := lockedReassignmentOwner{Owner: name, BeforeFiles: append([]string{}, prior.Previous.Owners[name]...), AfterFiles: []string{}, Symbols: []nameLockSymbol{}}
		if raw, exists := prior.Previous.NameLocks[name]; exists {
			entry.BeforeSHA256 = adoptionSHA256(raw)
		}
		if raw, exists := next.NameLocks[name]; exists {
			state, e := schema.ParseNameLock(raw, source.Profile.Package, source.Profile.Suffix, source.Profile.Qualified)
			if e != nil {
				return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
			}
			entry.AfterSHA256 = adoptionSHA256(raw)
			entry.Symbols = nameLockSymbols(state, generated[name].Files)
			for _, file := range generated[name].Files {
				entry.AfterFiles = append(entry.AfterFiles, file.Path)
			}
		}
		migration.Owners = append(migration.Owners, entry)
	}
	all.LockedOwnerReassignment = &migration
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end = timer.phase(phase)
	changed, e := output.WriteLockedOwnerReassignment(root, output.LockedOwnerReassignmentUpdate{Previous: prior, Generated: generated}, readonly)
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

func printLockedOwnerReassignment(w io.Writer, r lockedOwnerReassignmentReport) {
	_, _ = fmt.Fprintf(w, "locked owner reassignment (manifest v%d): SHA256 %s -> %s\n%s\n", r.Version, r.ManifestBeforeSHA256, r.ManifestAfterSHA256, r.Impact)
	_, _ = fmt.Fprintf(w, "  owners %s -> %s\n", strconv.Quote(fmt.Sprint(r.BeforeOwners)), strconv.Quote(fmt.Sprint(r.AfterOwners)))
	for _, file := range r.Files {
		_, _ = fmt.Fprintf(w, "  unchanged %s: %s\n", file.Path, file.SHA256)
	}
	for _, symbol := range r.SourceSymbols {
		_, _ = fmt.Fprintf(w, "  source %s %s descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
	}
	for _, owner := range r.Owners {
		_, _ = fmt.Fprintf(w, "  owner %s: files %s -> %s; lock SHA256 %s -> %s\n", owner.Owner, strconv.Quote(fmt.Sprint(owner.BeforeFiles)), strconv.Quote(fmt.Sprint(owner.AfterFiles)), owner.BeforeSHA256, owner.AfterSHA256)
		for _, symbol := range owner.Symbols {
			_, _ = fmt.Fprintf(w, "    %s %s descriptor=%s shape=%s occurrences=%s\n", symbol.Role, symbol.Classifier, symbol.Descriptor, symbol.Shape, strconv.Quote(fmt.Sprint(symbol.Occurrences)))
		}
	}
}
