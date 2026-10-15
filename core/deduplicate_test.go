package core_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestDeduplicateOperationEnvelope(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/deduplicate-models.json")
	if e != nil {
		t.Fatal(e)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		req := request(input, "Dto")
		req.Options.Validation = true
		req.Options.ModelStyle = style
		base, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		req.Options.DeduplicateModels = true
		r, e := core.GenerateWithDiagnostics(req)
		if e != nil {
			t.Fatal(e)
		}
		again, e := core.GenerateWithDiagnostics(req)
		if e != nil || !reflect.DeepEqual(r, again) || !reflect.DeepEqual(base.Warnings, r.Warnings) || len(r.Artifacts.Files) != len(base.Artifacts.Files) {
			t.Fatal("parity/determinism", e)
		}
		for _, pair := range []struct{ name, target string }{{"BetaResponseDto", "AlphaResponseDto"}, {"BetaInputDto", "AlphaInputDto"}, {"BetaModeDto", "AlphaModeDto"}, {"GuardInputDto", "AlphaInputDto"}} {
			text := source(t, r, pair.name)
			if !strings.Contains(text, "public typealias "+pair.name+" = org.example.models.api.models."+pair.target) {
				t.Fatal(pair, text)
			}
		}
		if strings.Contains(source(t, r, "GuardResponseDto"), "public typealias") {
			t.Fatal("different bound folded")
		}
		if strings.Contains(source(t, r, "AlphaResponseDto"), "public data class") == (style == core.PlainClass) {
			t.Fatal("style not applied after dedup")
		}
		req.Options.DeduplicateModels = false
		reset, e := core.GenerateWithDiagnostics(req)
		if e != nil || !reflect.DeepEqual(base, reset) {
			t.Fatal("false changed defaults", e)
		}
	}
}
func TestDeduplicateUniqueOperationOrderIndependent(t *testing.T) {
	input, e := os.ReadFile("../tests/fixtures/http/deduplicate-models.json")
	if e != nil {
		t.Fatal(e)
	}
	var d map[string]json.RawMessage
	if e = json.Unmarshal(input, &d); e != nil {
		t.Fatal(e)
	}
	var paths map[string]json.RawMessage
	if e = json.Unmarshal(d["paths"], &paths); e != nil {
		t.Fatal(e)
	}
	var reordered bytes.Buffer
	reordered.WriteString(`{`)
	for i, key := range []string{"/guard", "/beta", "/alpha"} {
		if i > 0 {
			reordered.WriteByte(',')
		}
		name, _ := json.Marshal(key)
		reordered.Write(name)
		reordered.WriteByte(':')
		reordered.Write(paths[key])
	}
	reordered.WriteByte('}')
	d["paths"] = reordered.Bytes()
	other, _ := json.Marshal(d)
	req := request(input, "Dto")
	req.Options.Validation = true
	req.Options.DeduplicateModels = true
	a, e := core.GenerateWithDiagnostics(req)
	if e != nil {
		t.Fatal(e)
	}
	req.Document = other
	b, e := core.GenerateWithDiagnostics(req)
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"AlphaInputDto", "BetaInputDto", "AlphaResponseDto", "BetaResponseDto", "AlphaModeDto", "BetaModeDto"} {
		if source(t, a, name) != source(t, b, name) {
			t.Fatal("canonical choice/order unstable", name)
		}
	}
}
func TestDeduplicateOperationSlotBudgets(t *testing.T) {
	for _, n := range []int{240, 241} {
		props := map[string]any{}
		required := []string{}
		for i := 0; i < n; i++ {
			name := fmt.Sprintf("f%03d", i)
			props[name] = map[string]any{"type": "integer", "format": "int64", "nullable": true}
			required = append(required, name)
		}
		shape := map[string]any{"type": "object", "properties": props, "required": required}
		paths := map[string]any{}
		for _, name := range []string{"alpha", "beta"} {
			paths["/"+name] = map[string]any{"get": map[string]any{"operationId": name, "responses": map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": shape}}}}}}
		}
		data, _ := json.Marshal(map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "Slot Contracts", "version": "1"}, "paths": paths})
		req := request(data, "Dto")
		a, e := core.GenerateWithDiagnostics(req)
		req.Options.DeduplicateModels = true
		b, f := core.GenerateWithDiagnostics(req)
		if n == 240 {
			if e != nil || f != nil || !strings.Contains(source(t, b, "BetaResponseDto"), "public typealias") {
				t.Fatal(n, e, f)
			}
		} else if e == nil || f == nil || e.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(e, f) || !reflect.DeepEqual(a, b) {
			t.Fatal("dedup bypassed original occurrence slots", e, f)
		}
	}
}
func TestDeduplicateBeforeAllOriginalBudgetFailures(t *testing.T) {
	for _, n := range []int{49, 50} {
		req := request(rootObjectChain(n, ""), "")
		req.Options.Validation = true
		a, e := core.GenerateWithDiagnostics(req)
		req.Options.DeduplicateModels = true
		b, f := core.GenerateWithDiagnostics(req)
		if e == nil || f == nil || !reflect.DeepEqual(e, f) || !reflect.DeepEqual(a, b) {
			t.Fatal("dedup bypassed source budget", e, f)
		}
	}
}
