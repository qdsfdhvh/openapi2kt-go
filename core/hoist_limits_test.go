package core_test

import (
	"encoding/json"
	"fmt"
	"openapi-kt/core"
	"reflect"
	"strings"
	"testing"
)

func hoistLimitDocument(kind string, n int) []byte {
	leaf := map[string]any{"type": "string", "minLength": 1}
	props := map[string]any{}
	required := []string{}
	switch kind {
	case "boxed", "long":
		scalar := any(leaf)
		if kind == "long" {
			scalar = map[string]any{"type": "integer", "format": "int64", "minimum": 1}
		}
		for i := 0; i < n; i++ {
			key := fmt.Sprintf("f%03d", i)
			props[key] = scalar
			required = append(required, key)
		}
	case "array":
		for i := 0; i < n; i++ {
			leaf = map[string]any{"type": "array", "items": leaf}
		}
		props["items"] = leaf
		required = append(required, "items")
	case "enum":
		props["id"] = map[string]any{"type": "string"}
		required = append(required, "id")
	}
	shape := map[string]any{"type": "object", "properties": props, "required": required}
	paths := map[string]any{}
	for _, owner := range []string{"alpha", "beta"} {
		op := map[string]any{"operationId": owner, "responses": map[string]any{"200": map[string]any{"content": map[string]any{"application/json": map[string]any{"schema": shape}}}}}
		if kind == "enum" {
			entries := []string{}
			for i := 0; i < n; i++ {
				entries = append(entries, fmt.Sprintf("v%03d", i))
			}
			op["parameters"] = []any{map[string]any{"name": "mode", "in": "query", "required": true, "schema": map[string]any{"type": "string", "enum": entries}}}
		}
		paths["/"+owner] = map[string]any{"get": op}
	}
	data, e := json.Marshal(map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "Hoist Limits", "version": "1"}, "paths": paths})
	if e != nil {
		panic(e)
	}
	return data
}
func TestHoistOriginalOccurrenceBudgets(t *testing.T) {
	for _, c := range []struct {
		kind  string
		limit int
	}{{"boxed", 240}, {"long", 120}, {"array", 16}, {"enum", 240}} {
		for _, n := range []int{c.limit, c.limit + 1} {
			data := hoistLimitDocument(c.kind, n)
			for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
				req := request(data, "Dto")
				req.Options.Validation = true
				req.Options.ModelStyle = style
				base, baseErr := core.GenerateWithDiagnostics(req)
				for _, qualify := range []bool{false, true} {
					for _, dedup := range []bool{false, true} {
						req.Options.HoistSharedShapes = true
						req.Options.QualifyHoistedNames = qualify
						req.Options.DeduplicateModels = dedup
						r, e := core.GenerateWithDiagnostics(req)
						if n == c.limit {
							if baseErr != nil || e != nil || !reflect.DeepEqual(base.Warnings, r.Warnings) {
								t.Fatal(c, n, qualify, dedup, baseErr, e)
							}
							if !strings.Contains(source(t, r, "BetaResponseDto"), "public typealias") {
								t.Fatal("root not hoisted", c)
							}
						} else if baseErr == nil || e == nil || !reflect.DeepEqual(baseErr, e) || !reflect.DeepEqual(base, r) {
							t.Fatal("hoist bypasses occurrence guard", c, n, baseErr, e)
						}
					}
				}
			}
		}
	}
	for _, n := range []int{49, 50} {
		req := request(rootObjectChain(n, ""), "")
		req.Options.Validation = true
		base, e := core.GenerateWithDiagnostics(req)
		req.Options.HoistSharedShapes = true
		req.Options.QualifyHoistedNames = true
		req.Options.DeduplicateModels = true
		r, f := core.GenerateWithDiagnostics(req)
		if e == nil || f == nil || !reflect.DeepEqual(e, f) || !reflect.DeepEqual(base, r) {
			t.Fatal("original depth failure lost", n, e, f)
		}
	}
}
