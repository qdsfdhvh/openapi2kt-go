package core

import (
	"sort"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/kotlin"
	"openapi-kt/internal/naming"
	"openapi-kt/internal/schema"
)

// Generate returns only artifacts. Consumers needing approximation warnings must
// use GenerateWithDiagnostics. Both functions produce identical files.
func Generate(request GenerationRequest) (GenerationArtifacts, error) {
	report, err := GenerateWithDiagnostics(request)
	if err != nil {
		return GenerationArtifacts{}, err
	}
	return report.Artifacts, nil
}

// GenerateWithDiagnostics generates Kotlin source entirely in memory. Errors
// return a zero report and a structured Diagnostics error, never partial files or
// warnings. Unsupported semantic constructs are errors, not silent projections.
func GenerateWithDiagnostics(request GenerationRequest) (GenerationReport, error) {
	return GenerateWithPhaseObserver(request, nil)
}

// GenerateWithPhaseObserver exposes actual core boundaries to a caller-owned
// observer. Timing/OS access remains the caller's responsibility; callbacks do
// not affect the generated report. See GenerationPhaseObserver for its contract.
func GenerateWithPhaseObserver(request GenerationRequest, observer GenerationPhaseObserver) (GenerationReport, error) {
	phases := phaseObserver{start: observer}
	defer phases.end()
	phases.begin(PhaseOptions)
	if err := naming.ValidateOptions(request.Options.PackageName, request.Options.ModelNameSuffix); err != nil {
		return GenerationReport{}, Diagnostics{*err}
	}
	if err := naming.ValidateClientOptions(request.Options.APIClientName, request.Options.ExcludeAPIs); err != nil {
		return GenerationReport{}, Diagnostics{*err}
	}
	if request.Options.RuntimePackage != "" {
		if err := naming.ValidateOptions(request.Options.RuntimePackage, ""); err != nil {
			return GenerationReport{}, Diagnostics{*err}
		}
	}
	if style := request.Options.ModelStyle; style != "" && style != DataClass && style != PlainClass {
		return GenerationReport{}, Diagnostics{*diagnostic.New("E_OPTIONS", "Unsupported model style.")}
	}
	if request.Options.QualifyHoistedNames && !request.Options.HoistSharedShapes {
		return GenerationReport{}, Diagnostics{*diagnostic.New("E_OPTIONS", "Hoisted name qualification requires shared-shape hoisting.")}
	}
	if request.Options.LockHoistedNames && !request.Options.HoistSharedShapes || !request.Options.LockHoistedNames && request.PreviousNameLock != nil {
		return GenerationReport{}, Diagnostics{*diagnostic.New("E_OPTIONS", "Hoisted name locking requires hoisting; previous state requires locking.")}
	}
	mergeExclusions, failure := naming.BuildMergeExclusions(request.Options.MergeIdenticalModels, request.Options.ExcludeMergeModels, request.Options.ExcludeMergeModelPatterns, request.Options.ExcludeMergeModelSuffixes)
	if failure != nil {
		return GenerationReport{}, Diagnostics{*failure}
	}
	if request.ProjectPreviousNameLock && (!request.Options.LockHoistedNames || request.PreviousNameLock == nil) {
		return GenerationReport{}, Diagnostics{*diagnostic.New("E_OPTIONS", "Name-lock projection requires explicit previous locked state.")}
	}
	var lock *schema.HoistedNameLock
	if request.Options.LockHoistedNames {
		var failure *diagnostic.Diagnostic
		lock, failure = schema.ParseNameLock(request.PreviousNameLock, request.Options.PackageName, request.Options.ModelNameSuffix, request.Options.QualifyHoistedNames)
		if failure != nil {
			return GenerationReport{}, Diagnostics{*failure}
		}
	}
	phases.begin(PhaseParse)
	root, err := document.Load(request.Document, request.Format, request.SourceName)
	if err != nil {
		return GenerationReport{}, Diagnostics{*err}
	}
	phases.begin(PhaseAnalyze)
	plan, warnings, err := schema.Analyze(root, schema.Options{PackageName: request.Options.PackageName, RuntimePackage: request.Options.RuntimePackage, Suffix: request.Options.ModelNameSuffix, ClientName: request.Options.APIClientName, ExcludeAPIs: request.Options.ExcludeAPIs, Validation: request.Options.Validation})
	if err != nil {
		return GenerationReport{}, Diagnostics{*err}
	}
	phases.begin(PhaseTransform)
	componentRoots := map[schema.Declaration]bool{}
	if request.Options.MergeIdenticalModels {
		for _, d := range plan.Models {
			componentRoots[d] = true
		}
	}
	if request.Options.HoistSharedShapes {
		if lock != nil {
			var failure *diagnostic.Diagnostic
			if request.ProjectPreviousNameLock {
				failure = schema.HoistOperationModelsLockedProjection(plan, request.Options.PackageName, request.Options.QualifyHoistedNames, lock)
			} else {
				failure = schema.HoistOperationModelsLocked(plan, request.Options.PackageName, request.Options.QualifyHoistedNames, lock)
			}
			if failure != nil {
				return GenerationReport{}, Diagnostics{*failure}
			}
		} else {
			schema.HoistOperationModels(plan, request.Options.PackageName, request.Options.QualifyHoistedNames)
		}
	}
	if request.Options.DeduplicateModels {
		schema.DeduplicateOperationModels(plan, request.Options.PackageName)
	}
	if request.Options.MergeIdenticalModels {
		schema.MergeIdenticalModels(plan, request.Options.PackageName, componentRoots, mergeExclusions.Matches)
	}
	if request.Options.ModelStyle == PlainClass {
		schema.UsePlainClasses(plan)
	}
	if failure := schema.CheckRetainedDataCopySignatureLimits(plan, request.Options.PackageName); failure != nil {
		return GenerationReport{}, Diagnostics{*failure}
	}
	if failure := schema.CheckRetainedEnumJVMNameLimits(plan); failure != nil {
		return GenerationReport{}, Diagnostics{*failure}
	}
	phases.begin(PhaseEmit)
	files := kotlin.Emit(plan.Models, request.Options.PackageName, plan.EmissionHints)
	if plan.Client != nil {
		files = append(files, kotlin.Emit(plan.Client.Models, request.Options.PackageName+".api", plan.EmissionHints)...)
	}
	if plan.PatchPackage != "" {
		files = append(files, kotlin.EmitPatch(plan.PatchPackage))
	}
	if plan.BigintPackage != "" {
		files = append(files, kotlin.EmitLongAsString(plan.BigintPackage))
	}
	files = append(files, kotlin.EmitClient(plan.Client, request.Options.PackageName)...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	artifacts := GenerationArtifacts{Files: make([]GeneratedFile, 0, len(files))}
	if lock != nil {
		var failure *diagnostic.Diagnostic
		artifacts.NameLock, failure = lock.Encode()
		if failure != nil {
			return GenerationReport{}, Diagnostics{*failure}
		}
	}
	for _, file := range files {
		artifacts.Files = append(artifacts.Files, GeneratedFile{Path: file.Path, Content: file.Content})
	}
	return GenerationReport{Artifacts: artifacts, Warnings: warnings}, nil
}
