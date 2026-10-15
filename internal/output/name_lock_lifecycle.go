package output

import (
	"openapi-kt/core"
	"openapi-kt/internal/diagnostic"
)

// NameLockAction explicitly replaces or removes existing manifest-owned state.
// It does not authorize profile changes or repair invalid output/recovery state.
type NameLockAction string

const (
	NameLockReset   NameLockAction = "reset"
	NameLockDisable NameLockAction = "disable"
)

// WriteNameLockLifecycle commits a reset or disable through the ordinary planned
// transaction, comparing expectedPrevious before and under the writer lock.
// Reset artifacts must carry valid state with the same profile. Disable artifacts
// have no state; callers must verify the generation profile against the old state
// before generating them, since unlocked artifacts do not encode that profile.
func WriteNameLockLifecycle(root string, artifacts core.GenerationArtifacts, expectedPrevious []byte, action NameLockAction, check bool) (bool, *diagnostic.Diagnostic) {
	if !validNameLockAction(action) {
		return false, fail("E_NAME_LOCK_LIFECYCLE", "Choose an explicit reset or disable name-lock action.")
	}
	return writePlanned(root, check, func(root string) (*plan, *diagnostic.Diagnostic) {
		return makeNameLockPlan(root, artifacts, expectedPrevious, action)
	})
}

func validNameLockAction(action NameLockAction) bool {
	return action == NameLockReset || action == NameLockDisable
}

func validateNameLockActionArtifacts(artifacts core.GenerationArtifacts, action NameLockAction) *diagnostic.Diagnostic {
	if action == "" {
		return nil
	}
	if !validNameLockAction(action) {
		return fail("E_NAME_LOCK_LIFECYCLE", "Choose an explicit reset or disable name-lock action.")
	}
	if len(artifacts.Files) == 0 || action == NameLockReset && artifacts.NameLock == nil || action == NameLockDisable && artifacts.NameLock != nil {
		return fail("E_NAME_LOCK_OUTPUT_INVALID", "Reset requires generated lock state; disable requires unlocked Kotlin output.")
	}
	return nil
}

func sameNameLockProfile(previous, next []byte) *diagnostic.Diagnostic {
	old, e := parseNameLockBytes(previous)
	if e != nil {
		return e
	}
	fresh, e := parseNameLockBytes(next)
	if e != nil {
		return e
	}
	if old.Profile != fresh.Profile {
		return fail("E_NAME_LOCK_PROFILE", "Name-lock lifecycle actions require the stored package, suffix and qualification profile.")
	}
	return nil
}
