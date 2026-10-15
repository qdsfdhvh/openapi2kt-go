package cli

import (
	"io"
	"openapi-kt/core"
	"openapi-kt/internal/config"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"openapi-kt/internal/output"
	"os"
	"path/filepath"
	"sort"
)

type specReport struct {
	Name     string           `json:"name"`
	Files    []string         `json:"files"`
	Warnings core.Diagnostics `json:"warnings,omitempty"`
}

func readGenerate(args arguments, timer *metricCollector) (core.GenerationReport, int, core.Diagnostics) {
	return readGeneratePrevious(args, timer, nil)
}
func readGeneratePrevious(args arguments, timer *metricCollector, previous []byte) (core.GenerationReport, int, core.Diagnostics) {
	// A lifecycle action regenerates from fresh allocation. Generate the same
	// fresh hoisted inventory for disable's report, then omit its persistence.
	if args.nameLockAction != "" {
		previous = nil
		args.lockHoistedNames = true
	}
	finishRead := timer.phase("read")
	reader := args.inputReader
	var owned *os.File
	if args.input != "-" {
		file, e := os.Open(args.input)
		if e != nil {
			finishRead()
			return core.GenerationReport{}, 4, failed("E_INPUT_IO", "Cannot read the input document.").Diagnostics
		}
		owned, reader = file, file
	}
	if reader == nil {
		finishRead()
		return core.GenerationReport{}, 4, failed("E_INPUT_IO", "Cannot read the input document.").Diagnostics
	}
	data, e := io.ReadAll(io.LimitReader(reader, document.MaxDocumentBytes+1))
	if owned != nil {
		_ = owned.Close()
	}
	finishRead()
	timer.input(len(data))
	if e != nil {
		return core.GenerationReport{}, 4, failed("E_INPUT_IO", "Cannot read the input document.").Diagnostics
	}
	finishGeneration := timer.phase("generate")
	generated, err := core.GenerateWithPhaseObserver(core.GenerationRequest{Document: data, Format: args.format, SourceName: args.input, PreviousNameLock: previous, Options: core.GenerationOptions{PackageName: args.packageName, RuntimePackage: args.runtimePackage, ModelStyle: args.modelStyle, DeduplicateModels: args.deduplicateModels, MergeIdenticalModels: args.mergeIdenticalModels, ExcludeMergeModels: args.excludeMergeModels, ExcludeMergeModelPatterns: args.excludeMergeModelPatterns, ExcludeMergeModelSuffixes: args.excludeMergeModelSuffixes, HoistSharedShapes: args.hoistSharedShapes, QualifyHoistedNames: args.qualifyHoistedNames, LockHoistedNames: args.lockHoistedNames, ModelNameSuffix: args.suffix, APIClientName: args.clientName, ExcludeAPIs: args.excludeAPIs, Validation: args.validation}}, timer.coreObserver())
	finishGeneration()
	if err != nil {
		return generated, 1, err.(core.Diagnostics)
	}
	return generated, 0, nil
}
func configArguments(base arguments, s config.Settings, dir string) arguments {
	a := arguments{command: base.command, format: core.Auto, nameLockAction: base.nameLockAction}
	if base.command == "validate" {
		a.packageName = "org.example.generated"
	}
	text := func(key string) string { v, _ := s[key].(string); return v }
	a.input = text("input")
	a.output = text("output_dir")
	if _, ok := s["package_name"]; ok {
		a.packageName = text("package_name")
	}
	a.runtimePackage = text("runtime_package")
	a.modelStyle = core.ModelStyle(text("model_style"))
	a.suffix = text("model_name_suffix")
	a.clientName = text("api_client_name")
	a.excludeAPIs, _ = s["exclude_apis"].([]string)
	a.validation, _ = s["validation"].(bool)
	a.deduplicateModels, _ = s["deduplicate_models"].(bool)
	a.mergeIdenticalModels, _ = s["merge_identical_models"].(bool)
	a.excludeMergeModels, _ = s["exclude_merge_models"].([]string)
	a.excludeMergeModelPatterns, _ = s["exclude_merge_model_patterns"].([]string)
	a.excludeMergeModelSuffixes, _ = s["exclude_merge_model_suffixes"].([]string)
	a.hoistSharedShapes, _ = s["hoist_shared_shapes"].(bool)
	a.qualifyHoistedNames, _ = s["qualify_hoisted_names"].(bool)
	a.lockHoistedNames, _ = s["lock_hoisted_names"].(bool)
	if _, ok := s["input_format"]; ok {
		a.format = core.InputFormat(text("input_format"))
	}
	if base.explicit["--package-name"] {
		a.packageName = base.packageName
	}
	if base.explicit["--hoist-shared-shapes"] {
		a.hoistSharedShapes = base.hoistSharedShapes
	}
	if base.explicit["--qualify-hoisted-names"] {
		a.qualifyHoistedNames = base.qualifyHoistedNames
	}
	if base.explicit["--lock-hoisted-names"] {
		a.lockHoistedNames = base.lockHoistedNames
	}
	if base.explicit["--merge-identical-models"] {
		a.mergeIdenticalModels = base.mergeIdenticalModels
	}
	if base.explicit["--exclude-merge-model"] {
		a.excludeMergeModels = base.excludeMergeModels
	}
	if base.explicit["--exclude-merge-model-pattern"] {
		a.excludeMergeModelPatterns = base.excludeMergeModelPatterns
	}
	if base.explicit["--exclude-merge-model-suffix"] {
		a.excludeMergeModelSuffixes = base.excludeMergeModelSuffixes
	}
	if base.explicit["--deduplicate-models"] {
		a.deduplicateModels = base.deduplicateModels
	}
	if base.explicit["--model-style"] {
		a.modelStyle = base.modelStyle
	}
	if base.explicit["--runtime-package"] {
		a.runtimePackage = base.runtimePackage
	}
	if base.explicit["--model-name-suffix"] {
		a.suffix = base.suffix
	}
	if base.explicit["--api-client-name"] {
		a.clientName = base.clientName
	}
	if base.explicit["--exclude-api"] {
		a.excludeAPIs = base.excludeAPIs
	}
	if base.explicit["--validation"] {
		a.validation = base.validation
	}
	if base.explicit["--input-format"] {
		a.format = base.format
	}
	if a.input != "" && !filepath.IsAbs(a.input) {
		a.input = filepath.Join(dir, a.input)
	}
	if a.output != "" && !filepath.IsAbs(a.output) {
		a.output = filepath.Join(dir, a.output)
	}
	if base.explicit["--output"] {
		a.output = base.output
		if absolute, e := filepath.Abs(a.output); e == nil {
			a.output = absolute
		}
	}
	return a
}
func executeConfig(args arguments, timer *metricCollector) (int, report) {
	end := timer.phase("config")
	finished := false
	finishConfig := func() {
		if !finished {
			end()
			finished = true
		}
	}
	defer finishConfig()
	timer.configBytes(0)
	path, e := filepath.Abs(args.config)
	if e != nil {
		return 4, failed("E_CONFIG_IO", "Cannot read the configuration.")
	}
	f, e := os.Open(path)
	if e != nil {
		return 4, failed("E_CONFIG_IO", "Cannot read the configuration.")
	}
	data, e := io.ReadAll(io.LimitReader(f, config.MaxBytes+1))
	_ = f.Close()
	timer.configBytes(len(data))
	if e != nil {
		return 4, failed("E_CONFIG_IO", "Cannot read the configuration.")
	}
	cfg, err := config.Parse(data)
	if err != nil {
		return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
	}
	dir := filepath.Dir(path)
	selected := 0
	for _, s := range cfg.Specs {
		if args.spec == "" || s.Name == args.spec {
			selected++
		}
	}
	if args.spec != "" && selected == 0 {
		return 1, failed("E_CONFIG_SPEC", "Selected spec does not exist in the configuration.")
	}
	overrides := false
	for _, key := range []string{"--output", "--package-name", "--runtime-package", "--model-style", "--deduplicate-models", "--merge-identical-models", "--exclude-merge-model", "--exclude-merge-model-pattern", "--exclude-merge-model-suffix", "--hoist-shared-shapes", "--qualify-hoisted-names", "--lock-hoisted-names", "--model-name-suffix", "--api-client-name", "--exclude-api", "--validation", "--input-format"} {
		overrides = overrides || args.explicit[key] && !(args.moveOutputFrom != "" && key == "--output")
	}
	if selected > 1 && overrides {
		return 1, failed("E_CONFIG_OVERRIDE", "Generation option overrides require a single selected spec.")
	}
	active := map[string][]string{}
	settings := map[string]arguments{}
	updates := map[string]output.OwnerUpdate{}
	previousLocks := map[string]map[string][]byte{}
	for _, s := range cfg.Specs {
		base := args
		if args.spec != "" && args.spec != s.Name {
			base.explicit = nil
			base.nameLockAction = ""
		}
		a := configArguments(base, config.Overlay(cfg.Defaults, s.Settings), dir)
		if err := naming.ValidateMergeExclusions(a.mergeIdenticalModels, a.excludeMergeModels, a.excludeMergeModelPatterns, a.excludeMergeModelSuffixes); err != nil {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: s.Name}
		}
		if a.lockHoistedNames && !a.hoistSharedShapes {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: failed("E_OPTIONS", "Hoisted name locking requires shared-shape hoisting.").Diagnostics, Spec: s.Name}
		}
		if !validNameLockAction(a) {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: failed("E_OPTIONS", "Name-lock reset requires locking; disable requires locking off; both require hoisting.").Diagnostics, Spec: s.Name}
		}
		if args.command == "validate" && (args.explicit["--output"] || args.explicit["--model-name-suffix"]) && (args.spec == "" || args.spec == s.Name) && !a.lockHoistedNames && a.nameLockAction == "" && len(args.ownerRenames) == 0 && args.moveOutputFrom == "" {
			return 2, failed("E_USAGE", "Validate output/suffix overrides require hoisted name locking.")
		}
		if a.qualifyHoistedNames && !a.hoistSharedShapes {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: failed("E_OPTIONS", "Hoisted name qualification requires shared-shape hoisting.").Diagnostics, Spec: s.Name}
		}
		if a.input == "" || a.packageName == "" || a.output == "" {
			return 1, failed("E_CONFIG_INVALID", "Each spec requires input, package_name and output_dir after defaults.")
		}
		if err := naming.ValidateOptions(a.packageName, a.suffix); err != nil {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: s.Name}
		}
		if a.runtimePackage != "" {
			if err := naming.ValidateOptions(a.runtimePackage, ""); err != nil {
				return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: s.Name}
			}
		}
		if err := naming.ValidateClientOptions(a.clientName, a.excludeAPIs); err != nil {
			return 1, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}, Spec: s.Name}
		}
		a.output = filepath.Clean(a.output)
		active[a.output] = append(active[a.output], s.Name)
		settings[s.Name] = a
	}
	if err := validateRuntimeRoots(settings); err != nil {
		return 1, failed("E_CONFIG_RUNTIME_ROOTS", "An explicit shared runtime package requires one declared output root across its configured specs.")
	}
	if len(cfg.Specs) == 0 {
		if args.nameLockAction != "" {
			return 1, failed("E_CONFIG_EMPTY", "Name-lock actions require selected specs with existing state.")
		}
		if !args.reconcile {
			return 1, failed("E_CONFIG_EMPTY", "An empty inventory requires explicit --reconcile.")
		}
		a := configArguments(args, cfg.Defaults, dir)
		if a.output == "" {
			return 1, failed("E_CONFIG_INVALID", "Empty reconciliation requires output_dir.")
		}
		updates[a.output] = output.OwnerUpdate{Reconcile: true}
	}
	finishConfig()
	if args.adoptLockedOwner {
		return executeLockedOwnerAdoption(args, cfg, settings, timer)
	}
	if args.moveOutputFrom != "" {
		return executeConfigRootMove(args, cfg, settings, dir, timer)
	}
	if len(args.ownerRenames) != 0 {
		return executeOwnerRename(args, cfg, settings, timer)
	}
	all := report{Status: "ok", Files: []string{}, Diagnostics: core.Diagnostics{}, Specs: []specReport{}}
	paths := map[string]string{}
	for _, s := range cfg.Specs {
		if args.spec != "" && args.spec != s.Name {
			continue
		}
		a := settings[s.Name]
		previous, code, diagnostics := readPreviousNameLock(a, s.Name, timer)
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: s.Name}
		}
		if a.lockHoistedNames || a.nameLockAction != "" {
			if previousLocks[a.output] == nil {
				previousLocks[a.output] = map[string][]byte{}
			}
			previousLocks[a.output][s.Name] = previous
		}
		generated, code, diagnostics := readGeneratePrevious(a, timer, previous)
		if code != 0 {
			return code, report{Status: "error", Files: []string{}, Diagnostics: diagnostics, Spec: s.Name}
		}
		if a.nameLockAction != "" {
			migration, e := nameLockMigrationReport(a, s.Name, previous, generated.Artifacts)
			if e != nil {
				return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*e}, Spec: s.Name}
			}
			all.NameLockMigrations = append(all.NameLockMigrations, migration)
			if a.nameLockAction == output.NameLockDisable {
				generated.Artifacts.NameLock = nil
			}
		}
		entry := specReport{Name: s.Name, Files: []string{}, Warnings: generated.Warnings}
		for _, file := range generated.Artifacts.Files {
			entry.Files = append(entry.Files, file.Path)
			paths[a.output+"\x00"+file.Path] = file.Path
		}
		all.Specs = append(all.Specs, entry)
		all.Warnings = append(all.Warnings, generated.Warnings...)
		update, exists := updates[a.output]
		if !exists {
			update = output.OwnerUpdate{ActiveOwners: active[a.output], Generated: map[string]core.GenerationArtifacts{}, Reconcile: args.spec == ""}
		}
		if args.adoptLegacy {
			update.AdoptLegacy = s.Name
		}
		update.Generated[s.Name] = generated.Artifacts
		updates[a.output] = update
	}
	timer.plannedConfig(updates)
	if args.denyWarnings {
		endPolicy := timer.phase("warning_policy")
		denied := deniedWarnings(all.Warnings, args.allowWarnings)
		endPolicy()
		if len(denied) != 0 {
			all.Status = "error"
			all.Diagnostics = denied
			all.NameLockMigrations = nil
			return 1, all
		}
	}
	if args.command == "generate" || args.nameLockAction != "" {
		phase := "commit"
		readonly := args.check || args.command == "validate"
		if readonly {
			phase = "check"
		}
		endOutput := timer.phase(phase)
		lockedUpdates := map[string]output.NameLockedOwnerUpdate{}
		for root, u := range updates {
			actions := map[string]output.NameLockAction{}
			for owner := range u.Generated {
				if settings[owner].nameLockAction != "" {
					actions[owner] = settings[owner].nameLockAction
				}
			}
			lockedUpdates[root] = output.NameLockedOwnerUpdate{OwnerUpdate: u, PreviousNameLocks: previousLocks[root], NameLockActions: actions}
		}
		changed, err := output.WriteNameLockedOwnersBatch(lockedUpdates, readonly)
		endOutput()
		if err != nil {
			return 4, report{Status: "error", Files: []string{}, Diagnostics: core.Diagnostics{*err}}
		}
		if args.check && changed {
			all.Status = "different"
		}
	}
	for _, path := range paths {
		all.Files = append(all.Files, path)
	}
	sort.Strings(all.Files)
	if all.Status == "different" {
		return 3, all
	}
	return 0, all
}
