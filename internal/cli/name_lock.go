package cli

import (
	"openapi-kt/core"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

// Lifecycle actions are invocation-only. Configuration stores the target mode,
// so a later ordinary run cannot accidentally repeat a reset.
func validNameLockAction(args arguments) bool {
	if args.nameLockAction == "" {
		return true
	}
	return args.hoistSharedShapes && (args.nameLockAction == output.NameLockReset && args.lockHoistedNames || args.nameLockAction == output.NameLockDisable && !args.lockHoistedNames)
}

// Selected state is a readonly snapshot, not a held output lock. The identical
// bytes go to the output adapter's under-lock CAS; lifecycle generation uses
// fresh state only after the owned old state has been validated.
func readPreviousNameLock(args arguments, owner string, timer *metricCollector) ([]byte, int, core.Diagnostics) {
	if !args.lockHoistedNames && args.nameLockAction == "" {
		return nil, 0, nil
	}
	finish := timer.phase("name_lock_read")
	defer finish()
	var data []byte
	var e *core.Diagnostic
	if owner == "" {
		data, e = output.ReadNameLock(args.output)
	} else {
		data, e = output.ReadOwnerNameLock(args.output, owner)
	}
	if e != nil {
		return nil, 4, core.Diagnostics{*e}
	}
	if args.nameLockAction != "" {
		if data == nil {
			return nil, 4, failed("E_NAME_LOCK_LIFECYCLE", "An explicit name-lock action requires existing owned state.").Diagnostics
		}
		if _, e := schema.ParseNameLock(data, args.packageName, args.suffix, args.qualifyHoistedNames); e != nil {
			return nil, 1, core.Diagnostics{*e}
		}
	}
	return data, 0, nil
}
