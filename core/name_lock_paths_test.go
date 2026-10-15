package core_test

import (
	"openapi-kt/core"
	"strings"
	"testing"
)

func TestNameLockStoredNameCannotCreateReservedPath(t *testing.T) {
	req := request(nameLockInput(t), "Dto")
	req.Options.Validation = true
	req.Options.HoistSharedShapes = true
	req.Options.LockHoistedNames = true
	state, e := core.Generate(req)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"Con", "AUX", "Nul", "COM1", "LPT9"} {
		req.PreviousNameLock = []byte(strings.Replace(string(state.NameLock), `"name":"InputDto"`, `"name":"`+name+`"`, 1))
		lockFailure(t, req, "E_NAME_LOCK_INVALID")
	}
}
func TestNameLockCannotBypassOriginalReservedRoleGuard(t *testing.T) {
	input := strings.ReplaceAll(string(hoistLimitDocument("enum", 2)), `"mode"`, `"con"`)
	for _, hoist := range []bool{false, true} {
		for _, lock := range []bool{false, true} {
			if lock && !hoist {
				continue
			}
			req := request([]byte(input), "")
			req.Options.HoistSharedShapes = hoist
			req.Options.LockHoistedNames = lock
			req.Options.Validation = true
			lockFailure(t, req, "E_NAME_INVALID")
		}
	}
}
