package cli

import (
	"crypto/sha256"
	"encoding/hex"

	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

type lockedAdoptionLock struct {
	BeforeSHA256 string                 `json:"before_sha256"`
	AfterSHA256  string                 `json:"after_sha256"`
	Profile      schema.NameLockProfile `json:"profile"`
	Symbols      []nameLockSymbol       `json:"symbols"`
}
type lockedOwnerAdoptionReport struct {
	FromOwner            string             `json:"from_owner,omitempty"`
	Owner                string             `json:"owner"`
	BeforeVersion        uint32             `json:"before_version"`
	AfterVersion         uint32             `json:"after_version"`
	ManifestBeforeSHA256 string             `json:"manifest_before_sha256"`
	ManifestAfterSHA256  string             `json:"manifest_after_sha256"`
	Files                []ownerRenameFile  `json:"files"`
	NameLock             lockedAdoptionLock `json:"name_lock"`
	Impact               string             `json:"impact"`
	SourceSymbols        []nameLockSymbol   `json:"source_symbols,omitempty"`
}

func adoptionSHA256(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

func executeLockedOwnerAdoption(args arguments, cfg config.Config, settings map[string]arguments, timer *metricCollector) (int, report) {
	if args.retainLockedOwner != "" {
		return executeRetainedLockedAdoption(args, cfg, settings, timer)
	}
	if args.adoptLockedOwnerFrom != "" {
		return executeExistingLockedAdoption(args, cfg, settings, timer)
	}
	if args.spec == "" {
		return executeSplitLockedAdoption(args, cfg, settings, timer)
	}
	if len(cfg.Specs) != 1 || args.spec == "" || cfg.Specs[0].Name != args.spec || !settings[args.spec].lockHoistedNames {
		return 4, failed("E_LOCKED_OWNER_ADOPTION_INVALID", "Locked ownership adoption requires one configured locked spec and its explicit --spec selection.")
	}
	a := settings[args.spec]
	end := timer.phase("name_lock_read")
	prior, e := output.ReadLockedOwnerAdoptionState(a.output, args.spec)
	end()
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}, Spec: args.spec}
	}
	generated, code, diagnostics := readGeneratePrevious(a, timer, prior.NameLock)
	if code != 0 {
		return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: args.spec}
	}
	timer.plannedConfig(map[string]output.OwnerUpdate{a.output: {Generated: map[string]core.GenerationArtifacts{args.spec: generated.Artifacts}}})
	if a.denyWarnings {
		end := timer.phase("warning_policy")
		denied := deniedWarnings(generated.Warnings, a.allowWarnings)
		end()
		if len(denied) != 0 {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: denied, Warnings: generated.Warnings}
		}
	}
	// Build the report from the pre-input state before any output mutation.
	state, e := schema.ParseNameLock(prior.NameLock, a.packageName, a.suffix, a.qualifyHoistedNames)
	if e != nil {
		return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	next, e := output.LockedOwnerAdoptionManifest(prior, args.spec)
	if e != nil {
		return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}}
	}
	hash := adoptionSHA256(prior.NameLock)
	migration := lockedOwnerAdoptionReport{Owner: args.spec, BeforeVersion: prior.Version, AfterVersion: 4, ManifestBeforeSHA256: adoptionSHA256(prior.Manifest), ManifestAfterSHA256: adoptionSHA256(next), Files: []ownerRenameFile{}, NameLock: lockedAdoptionLock{hash, hash, state.Profile, nameLockSymbols(state, prior.Files)}, Impact: "Only same-root ownership metadata and the owned lock filename change. Exact Kotlin bytes, public classifiers, descriptors and raw lock payload are preserved. Subsequent generation must use configuration with this named owner; direct version3 commands refuse version4. This point-in-time report is not an approval token, crash atomicity or automatic recovery guarantee."}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Warnings: generated.Warnings, Specs: []specReport{{Name: args.spec, Files: []string{}, Warnings: generated.Warnings}}, LockedOwnerAdoptions: []lockedOwnerAdoptionReport{migration}}
	hashes := fileHashes(prior.Files)
	for _, file := range prior.Files {
		all.Files = append(all.Files, file.Path)
		all.Specs[0].Files = append(all.Specs[0].Files, file.Path)
		all.LockedOwnerAdoptions[0].Files = append(all.LockedOwnerAdoptions[0].Files, ownerRenameFile{file.Path, hashes[file.Path]})
	}
	readonly := args.check || args.command == "validate"
	phase := "commit"
	if readonly {
		phase = "check"
	}
	end = timer.phase(phase)
	changed, e := output.WriteLockedOwnerAdoption(a.output, output.LockedOwnerAdoptionUpdate{Owner: args.spec, Previous: prior, Generated: generated.Artifacts}, readonly)
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
