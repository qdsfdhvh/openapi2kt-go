// Package core generates the accepted Kotlin model and HTTP subsets entirely
// in memory, including scalar/enum/const/composition/union and typed hierarchies.
// CLI IO and output transactions live in separate packages, not in the core.
package core

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

const MaxDocumentBytes = document.MaxDocumentBytes

type InputFormat = document.InputFormat

const (
	Auto = document.Auto
	JSON = document.JSON
	YAML = document.YAML
)

// ModelStyle chooses data-generated APIs or explicit value methods.
type ModelStyle string

const (
	DataClass  ModelStyle = "data_class"
	PlainClass ModelStyle = "plain_class"
)

// GenerationOptions controls Kotlin names and opt-in model validation.
type GenerationOptions struct {
	PackageName string
	// RuntimePackage relocates existing Patch/bigint helpers; empty preserves models.
	RuntimePackage    string
	ModelNameSuffix   string
	ModelStyle        ModelStyle
	DeduplicateModels bool
	// MergeIdenticalModels folds eligible resolved ordinary model shapes; opt-in.
	MergeIdenticalModels bool
	// ExcludeMergeModels are exact fully qualified generated root classifiers.
	ExcludeMergeModels []string
	// Patterns match whole generated fully qualified roots; suffixes match the
	// short generated classifier including ModelNameSuffix. Both require merge.
	ExcludeMergeModelPatterns []string
	ExcludeMergeModelSuffixes []string
	HoistSharedShapes         bool
	QualifyHoistedNames       bool
	// LockHoistedNames emits/reuses pure-core versioned state; requires hoisting.
	LockHoistedNames bool
	APIClientName    string
	ExcludeAPIs      []string
	Validation       bool
}

// GenerationRequest is an in-memory input. SourceName is used only for format
// detection, never for embedding paths in generated source or diagnostics.
// Callers must not mutate Document or option slices concurrently with generation.
type GenerationRequest struct {
	Document   []byte
	Format     InputFormat
	SourceName string
	Options    GenerationOptions
	// PreviousNameLock is nil for a fresh lock; nonnil state is strictly validated.
	PreviousNameLock []byte
	// ProjectPreviousNameLock is an explicit migration-planning mode. Retain
	// original allocations for current origins, including singleton groups,
	// without allocating new locked shapes. Callers must separately verify whole
	// ownership/file/occurrence coverage before committing projected metadata.
	ProjectPreviousNameLock bool
}

// GeneratedFile has a validated slash-separated relative output path.
type GeneratedFile struct {
	Path    string
	Content string
}

type GenerationArtifacts struct {
	Files []GeneratedFile
	// NameLock is pure metadata, not a Kotlin file or a filesystem transaction.
	NameLock []byte
}

// GenerationReport adds non-fatal analysis warnings to files-only artifacts.
// A generation failure must return no partial report, artifacts, or warnings.
type GenerationReport struct {
	Artifacts GenerationArtifacts
	Warnings  []Diagnostic
}

type Diagnostic = diagnostic.Diagnostic
type Diagnostics = diagnostic.Diagnostics
