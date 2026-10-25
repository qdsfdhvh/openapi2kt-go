package cli

import (
	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

func executeSplitLockedAdoption(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	invalid := func() (int, report) {
		return 4, failed("E_LOCKED_OWNER_ADOPTION_INVALID", "Split adoption requires a complete nonempty locked configuration in one existing root.")
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
	prior, err := output.ReadSplitLockedAdoptionState(root, names)
	end()
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
	}
	budget, err := output.NewSplitAdoptionBudget(prior.Files)
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
	}
	generated := map[string]core.GenerationArtifacts{}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Specs: []specReport{}, LockedOwnerAdoptions: []lockedOwnerAdoptionReport{}}
	for _, spec := range cfg.Specs {
		a := settings[spec.Name]
		a.projectNameLock = prior.Version == 3
		g, code, diagnostics := readGeneratePrevious(a, timer, prior.NameLocks[spec.Name])
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: spec.Name}
		}
		if err := budget.Add(spec.Name, g.Artifacts); err != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: spec.Name}
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
	next, err := output.PlanSplitLockedAdoption(prior, generated)
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
	}
	originalSymbols := []nameLockSymbol{}
	if prior.Version == 3 {
		state, e := schema.ParseNameLock(prior.NameLock, settings[cfg.Specs[0].Name].packageName, settings[cfg.Specs[0].Name].suffix, settings[cfg.Specs[0].Name].qualifyHoistedNames)
		if e != nil {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
		}
		originalSymbols = nameLockSymbols(state, prior.Files)
	}
	for i, spec := range cfg.Specs {
		a := settings[spec.Name]
		payload := next.NameLocks[spec.Name]
		state, e := schema.ParseNameLock(payload, a.packageName, a.suffix, a.qualifyHoistedNames)
		if e != nil {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
		}
		before := prior.NameLocks[spec.Name]
		migration := lockedOwnerAdoptionReport{Owner: spec.Name, BeforeVersion: prior.Version, AfterVersion: 4, ManifestBeforeSHA256: adoptionSHA256(prior.Manifest), ManifestAfterSHA256: adoptionSHA256(next.Manifest), Files: []ownerRenameFile{}, NameLock: lockedAdoptionLock{adoptionSHA256(before), adoptionSHA256(payload), state.Profile, nameLockSymbols(state, generated[spec.Name].Files)}, Impact: "The complete original Kotlin inventory is covered byte-for-byte by the named owners. Shared files retain multiple memberships; projected locks preserve original allocation names and occurrence union. Only ownership and lock metadata change in this same-root transaction. Use ordinary named configuration afterward; no Kotlin writes, allocation reset, crash atomicity or automatic recovery."}
		if i == 0 {
			migration.SourceSymbols = originalSymbols
		}
		hashes := fileHashes(generated[spec.Name].Files)
		for _, file := range generated[spec.Name].Files {
			migration.Files = append(migration.Files, ownerRenameFile{file.Path, hashes[file.Path]})
		}
		all.LockedOwnerAdoptions = append(all.LockedOwnerAdoptions, migration)
	}
	for _, file := range prior.Files {
		all.Files = append(all.Files, file.Path)
	}
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end = timer.phase(phase)
	changed, err := output.WriteSplitLockedAdoption(root, output.SplitLockedAdoptionUpdate{Previous: prior, Generated: generated}, readonly)
	end()
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
	}
	if args.check && changed {
		all.Status = "different"
		return 3, all
	}
	return 0, all
}
