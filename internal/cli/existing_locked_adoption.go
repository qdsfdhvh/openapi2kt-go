package cli

import (
	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

func executeExistingLockedAdoption(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	a, ok := settings[args.spec]
	if !ok || args.spec == "" || !a.lockHoistedNames {
		return 4, failed("E_LOCKED_OWNER_ADOPTION_INVALID", "Existing-owner adoption requires an explicit locked --spec and donor.")
	}
	end := timer.phase("name_lock_read")
	prior, err := output.ReadExistingLockedAdoptionState(a.output, args.spec, args.adoptLockedOwnerFrom)
	end()
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: args.spec}
	}
	p := prior.Previous
	// Parse the effective donor profile before opening the input. Projection
	// retains only original allocations/occurrences consumed by the new spec.
	state, err := schema.ParseNameLock(p.NameLocks[prior.Donor], a.packageName, a.suffix, a.qualifyHoistedNames)
	if err != nil {
		return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: args.spec}
	}
	a.projectNameLock = true
	g, code, diagnostics := readGeneratePrevious(a, timer, p.NameLocks[prior.Donor])
	if code != 0 {
		return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: args.spec}
	}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Warnings: g.Warnings, Specs: []specReport{{Name: args.spec, Files: []string{}, Warnings: g.Warnings}}}
	timer.plannedConfig(map[string]output.OwnerUpdate{a.output: {Generated: map[string]core.GenerationArtifacts{args.spec: g.Artifacts}}})
	if denied := configDeniedWarnings(all.Specs, settings, timer); len(denied) != 0 {
		all.Status, all.Diagnostics = "error", denied
		return 1, all
	}
	next, err := output.PlanExistingLockedAdoption(prior, g.Artifacts)
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: args.spec}
	}
	payload := next.NameLocks[args.spec]
	newState, err := schema.ParseNameLock(payload, a.packageName, a.suffix, a.qualifyHoistedNames)
	if err != nil {
		return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: args.spec}
	}
	before := p.NameLocks[args.spec]
	migration := lockedOwnerAdoptionReport{Owner: args.spec, FromOwner: prior.Donor, BeforeVersion: 4, AfterVersion: 4, ManifestBeforeSHA256: adoptionSHA256(p.Manifest), ManifestAfterSHA256: adoptionSHA256(next.Manifest), Files: []ownerRenameFile{}, NameLock: lockedAdoptionLock{adoptionSHA256(before), adoptionSHA256(payload), newState.Profile, nameLockSymbols(newState, g.Artifacts.Files)}, SourceSymbols: nameLockSymbols(state, p.Files), Impact: "Adds an unchanged subset of the explicit donor's Kotlin files as shared ownership. Existing memberships and raw locks remain unchanged. Only the new owner lock and manifest are written; use ordinary named configuration afterward. Replay requires the same membership, lock semantics and donor allocation subset. This point-in-time report is not a crash-atomicity or automatic-recovery guarantee."}
	hashes := fileHashes(g.Artifacts.Files)
	for _, file := range g.Artifacts.Files {
		all.Files = append(all.Files, file.Path)
		all.Specs[0].Files = append(all.Specs[0].Files, file.Path)
		migration.Files = append(migration.Files, ownerRenameFile{file.Path, hashes[file.Path]})
	}
	all.LockedOwnerAdoptions = []lockedOwnerAdoptionReport{migration}
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end = timer.phase(phase)
	changed, err := output.WriteExistingLockedAdoption(a.output, output.ExistingLockedAdoptionUpdate{Previous: prior, Generated: g.Artifacts}, readonly)
	end()
	if err != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: args.spec}
	}
	if args.check && changed {
		all.Status = "different"
		return 3, all
	}
	return 0, all
}
