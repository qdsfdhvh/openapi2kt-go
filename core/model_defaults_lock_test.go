package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestModelDefaultsHoistedLockReplayAndShapeChange(t *testing.T) {
	operation := func(id, value string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"value":{"type":"string","default":"` + value + `"}}}}}}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + operation("alpha", "ready") + `,"/b":` + operation("beta", "ready") + `}}`)
	rq := request(doc, "Dto")
	rq.Options.HoistSharedShapes = true
	rq.Options.LockHoistedNames = true
	rq.Options.DeduplicateModels = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal(got, err)
	}
	rq.PreviousNameLock = got.Artifacts.NameLock
	again, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("default lock replay", again, err)
	}
	rq.Document = []byte(strings.ReplaceAll(string(doc), `"default":"ready"`, `"default":"changed"`))
	changed, err := core.GenerateWithDiagnostics(rq)
	if err == nil || len(changed.Artifacts.Files) != 0 || len(changed.Artifacts.NameLock) != 0 || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
		t.Fatal("changed default reused locked shape", changed, err)
	}
}
