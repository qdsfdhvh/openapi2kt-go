package core_test

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"openapi-kt/core"
)

func TestGenerationPhaseObserverReportsEnteredBoundariesAndIdenticalArtifacts(t *testing.T) {
	request := core.GenerationRequest{Document: []byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"id":{"type":"integer"}}}}}}`), Options: core.GenerationOptions{PackageName: "org.example.phases"}}
	base, err := core.GenerateWithDiagnostics(request)
	if err != nil {
		t.Fatal(err)
	}
	var events []string
	observer := func(phase core.GenerationPhase) func() {
		events = append(events, "start:"+string(phase))
		return func() { events = append(events, "end:"+string(phase)) }
	}
	measured, err := core.GenerateWithPhaseObserver(request, observer)
	if err != nil || !reflect.DeepEqual(base, measured) {
		t.Fatal(err, measured)
	}
	want := []string{"start:options", "end:options", "start:parse", "end:parse", "start:analyze", "end:analyze", "start:transform", "end:transform", "start:emit", "end:emit"}
	if !reflect.DeepEqual(events, want) {
		t.Fatal(events)
	}
	for _, tc := range []struct {
		name   string
		change func(*core.GenerationRequest)
		last   core.GenerationPhase
	}{
		{"options", func(r *core.GenerationRequest) { r.Options.PackageName = "" }, core.PhaseOptions},
		{"parse", func(r *core.GenerationRequest) { r.Document = []byte(`{"openapi":`) }, core.PhaseParse},
		{"analyze", func(r *core.GenerationRequest) {
			r.Document = []byte(`{"openapi":"3.1.0","components":{"schemas":{"Bad":{"type":"string","default":"x"}}}}`)
		}, core.PhaseAnalyze},
		{"lock_parse", func(r *core.GenerationRequest) {
			r.Options.HoistSharedShapes = true
			r.Options.LockHoistedNames = true
			r.Options.QualifyHoistedNames = true
			r.PreviousNameLock = []byte(`{}`)
		}, core.PhaseOptions},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request
			tc.change(&r)
			events = nil
			zero, err := core.GenerateWithPhaseObserver(r, observer)
			plain, plainErr := core.GenerateWithDiagnostics(r)
			if err == nil || !reflect.DeepEqual(zero, core.GenerationReport{}) || !reflect.DeepEqual(zero, plain) || !reflect.DeepEqual(err, plainErr) {
				t.Fatal(zero, err, plainErr)
			}
			if events[len(events)-1] != "end:"+string(tc.last) {
				t.Fatal(events)
			}
		})
	}
	// A valid persisted lock with semantic drift reaches Transform, closes it on
	// failure, and still returns no partial warnings/files/state.
	locked := request
	locked.Document = nameLockInput(t)
	locked.Options.HoistSharedShapes = true
	locked.Options.LockHoistedNames = true
	locked.Options.Validation = true
	initial, err := core.GenerateWithDiagnostics(locked)
	if err != nil {
		t.Fatal(err)
	}
	locked.PreviousNameLock = initial.Artifacts.NameLock
	locked.Document = []byte(strings.Replace(string(locked.Document), `"minLength": 2`, `"minLength": 3`, 1))
	events = nil
	zero, err := core.GenerateWithPhaseObserver(locked, observer)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(zero, core.GenerationReport{}) || !reflect.DeepEqual(events, want[:8]) {
		t.Fatal(err, zero, events)
	}
	// An observer may return nil; disabled/nil callbacks remain valid.
	got, err := core.GenerateWithPhaseObserver(request, func(core.GenerationPhase) func() { return nil })
	if err != nil || !reflect.DeepEqual(got, base) {
		t.Fatal(err, got)
	}
	// Observation is invocation-local; the same immutable request can be
	// generated concurrently with independent caller-owned collectors.
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			var local []string
			r, e := core.GenerateWithPhaseObserver(request, func(p core.GenerationPhase) func() {
				local = append(local, "start:"+string(p))
				return func() { local = append(local, "end:"+string(p)) }
			})
			if e != nil || !reflect.DeepEqual(r, base) || !reflect.DeepEqual(local, want) {
				t.Error(e, local)
			}
		})
	}
	workers.Wait()
}
