package core_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestBorrowedCheckedStringEnumSourceBeforeEmptyAndNull(t *testing.T) {
	for _, tc := range []struct{ checks, code, tail string }{
		{`"minLength":-1`, "E_SCHEMA_INVALID", "/minLength"},
		{`"minLength":1.0`, "E_SCHEMA_INVALID", "/minLength"},
		{`"minLength":3,"maxLength":1`, "E_SCHEMA_INVALID", "/maxLength"},
		{`"pattern":"["`, "E_SCHEMA_INVALID", "/pattern"},
		{`"pattern":"(?=x)"`, "E_SCHEMA_UNSUPPORTED", "/pattern"},
	} {
		for _, value := range []string{`[]`, `null`} {
			doc := schemas(`{"ZOwner":{"properties":{"word":{"enum":["one","x"],` + tc.checks + `}}},"Alias":{"$ref":"#/components/schemas/ZOwner/properties/word"},"Use":{"properties":{"rows":{"type":"array","nullable":true,"items":{"$ref":"#/components/schemas/Alias"},"default":` + value + `}}}}`)
			enumBoundsDiagnostic(t, doc, true, tc.code, "/components/schemas/ZOwner/properties/word"+tc.tail)
		}
	}
}

func TestBorrowedCheckedStringEnumAliasAndDefaultBudgets(t *testing.T) {
	for _, hops := range []int{48, 49} {
		parts := []string{`"Owner":{"properties":{"word":{"enum":["one","x"],"minLength":2}}}`}
		for i := 0; i < hops; i++ {
			target := fmt.Sprintf("A%02d", i+1)
			if i == hops-1 {
				target = "Owner/properties/word"
			}
			parts = append(parts, fmt.Sprintf(`"A%02d":{"$ref":"#/components/schemas/%s"}`, i, target))
		}
		rq := request(schemas(`{`+strings.Join(parts, ",")+`}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if hops == 48 && err != nil {
			t.Fatal("checked nominal alias rejected original48 hops", err)
		}
		if hops == 49 && (err == nil || err.(core.Diagnostics)[0].Code != "E_SCHEMA_DEPTH" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("checked nominal alias escaped original48 hops", err)
		}
	}
	for _, size := range []int{32700, 32768} {
		word := strings.Repeat("a", size)
		rq := request(schemas(`{"Owner":{"properties":{"word":{"enum":["`+word+`"],"minLength":1}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/word","default":"`+word+`"}}`), "Dto")
		rq.Options.Validation = true
		got, err := core.GenerateWithDiagnostics(rq)
		if size == 32700 && err != nil {
			t.Fatal("checked alias refused valid default below32KiB", err)
		}
		if size == 32768 && (err == nil || err.(core.Diagnostics)[0].Code != "E_MODEL_LIMIT" || !reflect.DeepEqual(got, core.GenerationReport{})) {
			t.Fatal("checked alias escaped default32KiB", err)
		}
	}
}

func TestBorrowedCheckedStringEnumFrozenCheckIdentity(t *testing.T) {
	shape := `{"properties":{"word":{"$ref":"#/components/schemas/Alias","default":"one"}}}`
	op := func(id string) string {
		return `{"get":{"operationId":"` + id + `","responses":{"200":{"content":{"application/json":{"schema":` + shape + `}}}}}}`
	}
	doc := `{"openapi":"3.1.0","components":{"schemas":{"Owner":{"properties":{"word":{"enum":["one","two","x"],"minLength":2}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/word"}}},"paths":{"/a":` + op("alpha") + `,"/b":` + op("beta") + `}}`
	rq := request([]byte(doc), "Dto")
	rq.Options.Validation, rq.Options.HoistSharedShapes, rq.Options.LockHoistedNames = true, true, true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil || len(got.Artifacts.NameLock) == 0 {
		t.Fatal("checked lexical reference lost frozen shape", err)
	}
	rq.PreviousNameLock = got.Artifacts.NameLock
	rq.Document = []byte(strings.Replace(doc, `"minLength":2`, `"minLength":3`, 1))
	changed, err := core.GenerateWithDiagnostics(rq)
	if err == nil || err.(core.Diagnostics)[0].Code != "E_NAME_LOCK_CHANGED" || !reflect.DeepEqual(changed, core.GenerationReport{}) {
		t.Fatal("checked lexical source reused changed frozen checks", err)
	}
}

func TestBorrowedCheckedStringEnumNullablePresenceAndStrictDefaults(t *testing.T) {
	body := `{"ZOwner":{"properties":{"word":{"enum":["one","two","x"],"nullable":true,"minLength":2,"maxLength":6,"default":"one"}}},"Alias":{"$ref":"#/components/schemas/ZOwner/properties/word","nullable":false,"default":"two"},"Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},"Use":{"properties":{"word":{"$ref":"#/components/schemas/Alias","default":"one"},"rows":{"type":"array","uniqueItems":true,"items":{"$ref":"#/components/schemas/Maybe"},"default":["one",null]}},"required":["word"]}}`
	rq := request(schemas(body), "Dto")
	rq.Options.Validation = true
	got, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal("checked source nearest nullable/default/Patch roles", err)
	}
	if strings.Contains(source(t, got, "UseDto"), "public val word:") && strings.Contains(source(t, got, "UseDto"), "public val word: org.example.models.ZOwnerDto.Word =") {
		t.Fatal("required checked reference inherited initializer")
	}
	for _, tc := range []struct{ from, to, pointer string }{
		{`"default":"two"`, `"default":null`, "/components/schemas/Alias/default"},
		{`"default":"two"`, `"default":"future"`, "/components/schemas/Alias/default"},
		{`"default":["one",null]`, `"default":["one","one"]`, "/components/schemas/Use/properties/rows/default/1"},
		{`"default":["one",null]`, `"default":["x",null]`, "/components/schemas/Use/properties/rows/default/0"},
	} {
		enumBoundsDiagnostic(t, schemas(strings.Replace(body, tc.from, tc.to, 1)), true, "E_SCHEMA_INVALID", tc.pointer)
	}
}

func TestBorrowedCheckedStringEnumRemainingSourceRoles(t *testing.T) {
	// A reference to the actual checked leaf may now own its own item
	// annotation; the leaf's original bounds still validate that value.
	child := `{"Owner":{"properties":{"word":{"enum":["one","x"],"minLength":2}}},"Tree":{"type":"array","items":{"$ref":"#/components/schemas/Owner/properties/word","default":"one"}}}`
	rq := request(schemas(child), "Dto")
	rq.Options.Validation = true
	independentEnumFactoryGraduation(t, rq)
	enumBoundsDiagnostic(t, schemas(strings.Replace(child, `"default":"one"`, `"default":"x"`, 1)), true, "E_SCHEMA_INVALID", "/components/schemas/Tree/items/default")
	body := `{"Owner":{"properties":{"rows":{"type":"array","items":{"enum":["one","x"],"minLength":2,"default":"one"}}}},"Alias":{"$ref":"#/components/schemas/Owner/properties/rows/items"}}`
	rq = request(schemas(body), "Dto")
	rq.Options.Validation = true
	inlineEnumChildGraduation(t, rq, `,"default":"one"`, `,"default":"x"`, "/components/schemas/Owner/properties/rows/items/default")
}
