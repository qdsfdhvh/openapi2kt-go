package output

import (
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// SplitAdoptionBudget bounds retained candidates before adapters accumulate the
// next owner. It counts shared bytes per candidate and conservatively bounds
// proposed ownership JSON, so empty files cannot evade membership limits.
type SplitAdoptionBudget struct{ kotlinBytes, lockBytes, manifestBytes int }

func NewSplitAdoptionBudget(original []core.GeneratedFile) (*SplitAdoptionBudget, *diagnostic.Diagnostic) {
	b := &SplitAdoptionBudget{manifestBytes: 1024}
	for _, file := range original {
		if len(file.Path)+80 > maxFileBytes-b.manifestBytes {
			return nil, fail("E_OUTPUT_LIMIT", "Split adoption ownership metadata exceeds the 64 MiB limit.")
		}
		b.manifestBytes += len(file.Path) + 80
	}
	return b, nil
}

func (b *SplitAdoptionBudget) Add(owner string, a core.GenerationArtifacts) *diagnostic.Diagnostic {
	if 2*len(owner)+160 > maxFileBytes-b.manifestBytes {
		return fail("E_OUTPUT_LIMIT", "Split adoption ownership metadata exceeds the 64 MiB limit.")
	}
	b.manifestBytes += 2*len(owner) + 160
	for _, file := range a.Files {
		if len(file.Content) > maxAdoptionKotlinBytes-b.kotlinBytes {
			return fail("E_OUTPUT_LIMIT", "Split adoption candidate Kotlin exceeds the 64 MiB aggregate limit.")
		}
		b.kotlinBytes += len(file.Content)
		if len(file.Path)+24 > maxFileBytes-b.manifestBytes {
			return fail("E_OUTPUT_LIMIT", "Split adoption ownership metadata exceeds the 64 MiB limit.")
		}
		b.manifestBytes += len(file.Path) + 24
	}
	if len(a.NameLock) > maxOwnerNameLockBytes-b.lockBytes {
		return fail("E_NAME_LOCK_LIMIT", "Named name-lock inventory exceeds metadata limits.")
	}
	b.lockBytes += len(a.NameLock)
	return nil
}
