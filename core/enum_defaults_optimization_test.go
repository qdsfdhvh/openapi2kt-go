package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestEnumDefaultsOptimizationAndLockIdentity(t *testing.T) {
	operation := func(id, value string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":{"properties":{"state":{"enum":["a","b"],"default":"` + value + `"}}}}}}}}}`
	}
	doc := []byte(`{"openapi":"3.1.0","paths":{"/a":` + operation("alpha", "a") + `,"/b":` + operation("beta", "a") + `,"/c":` + operation("gamma", "b") + `}}`)
	rq := request(doc, "Dto")
	rq.Options.HoistSharedShapes = true
	rq.Options.LockHoistedNames = true
	rq.Options.DeduplicateModels = true
	rq.Options.MergeIdenticalModels = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, got, "AlphaResponseDto"), "typealias") || strings.Contains(source(t, got, "GammaResponseDto"), "typealias") {
		t.Fatal("default identity folded", got)
	}
	text := source(t, got, "ResponseDto")
	if !strings.Contains(text, "private fun `openapi-kt default enum org example models models ResponseDto State`") || !strings.Contains(text, "org.example.models.models.ResponseDto.State.fromSerialName(value)") {
		t.Fatal("hoisted lexical helper invalid", text)
	}
	rq.PreviousNameLock = got.Artifacts.NameLock
	again, err := core.GenerateWithDiagnostics(rq)
	if err != nil || !reflect.DeepEqual(got, again) {
		t.Fatal("enum default lock replay", again, err)
	}
	rq.Document = []byte(strings.ReplaceAll(string(doc), `"default":"a"`, `"default":"b"`))
	changed, err := core.GenerateWithDiagnostics(rq)
	if err == nil || len(changed.Artifacts.Files) != 0 || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" {
		t.Fatal("changed enum default lock admitted", changed, err)
	}
}
