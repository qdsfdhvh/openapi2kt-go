package core_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestMergePrimitiveUnionRoots(t *testing.T) {
	input := schemas(`{"A":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Z":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Different":{"oneOf":[{"type":"string"},{"type":"boolean"}]},"Holder":{"properties":{"a":{"$ref":"#/components/schemas/A"},"z":{"$ref":"#/components/schemas/Z"}},"required":["a","z"]}}`)
	rq := request(input, "Dto")
	rq.Options.PackageName = "org.example.unionmerge"
	before, err := core.GenerateWithDiagnostics(rq)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source(t, before, "ZDto"), "sealed class ZDto") {
		t.Fatal("merge off changed union root")
	}
	original := bytes.Clone(input)
	rq.Options.MergeIdenticalModels = true
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		rq.Options.ModelStyle = style
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(source(t, got, "ZDto"), "public typealias ZDto = org.example.unionmerge.models.ADto") || !strings.Contains(source(t, got, "DifferentDto"), "sealed class DifferentDto") {
			t.Fatal("equal unions not merged or different kinds collapsed", source(t, got, "ZDto"))
		}
		for _, kind := range []string{"String", "Int"} {
			if !strings.Contains(source(t, got, "ZDto"), "public typealias "+kind+"ZDto = org.example.unionmerge.models."+kind+"ADto") {
				t.Fatal("lost public sibling wrapper constructor", kind)
			}
		}
		if !strings.Contains(source(t, got, "HolderDto"), "val z: org.example.unionmerge.models.ADto") {
			t.Fatal("typed union reference not rebound")
		}
		if !reflect.DeepEqual(before.Warnings, got.Warnings) || !bytes.Equal(original, input) {
			t.Fatal("merge altered source diagnostics")
		}
		again, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !reflect.DeepEqual(got, again) {
			t.Fatal("union merge nondeterministic", err)
		}
	}
}

func TestMergePrimitiveUnionGuards(t *testing.T) {
	for _, tc := range []struct{ extra, different string }{
		{`"nullable":true,`, ""},
		{"", `{"type":"integer"},{"type":"string"}`},
	} {
		other := tc.different
		if other == "" {
			other = `{"type":"string"},{"type":"integer"}`
		}
		input := schemas(`{"A":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Z":{` + tc.extra + `"oneOf":[` + other + `]}}`)
		rq := request(input, "Dto")
		rq.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, got, "ZDto"), "sealed class ZDto") {
			t.Fatal("null policy or ordered kinds collapsed", tc, err)
		}
	}
	for _, excluded := range []string{"ADto", "ZDto"} {
		input := schemas(`{"A":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Z":{"oneOf":[{"type":"string"},{"type":"integer"}]}}`)
		rq := request(input, "Dto")
		rq.Options.PackageName = "org.example.unionmerge"
		rq.Options.MergeIdenticalModels = true
		rq.Options.ExcludeMergeModels = []string{"org.example.unionmerge.models." + excluded}
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, got, "ZDto"), "sealed class ZDto") {
			t.Fatal("union exclusion absorbed or was absorbed", excluded, err)
		}
	}
	for _, token := range []string{"ZDto", "StringZDto", "ZDtoSerializer", "StringZDtoSerializer"} {
		input := schemas(`{"A":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Z":{"oneOf":[{"type":"string"},{"type":"integer"}]},"Provider":{"type":"string","x-native-type":"org.example.unionmerge.models.` + token + `"}}`)
		rq := request(input, "Dto")
		rq.Options.PackageName = "org.example.unionmerge"
		rq.Options.MergeIdenticalModels = true
		got, err := core.GenerateWithDiagnostics(rq)
		if err != nil || !strings.Contains(source(t, got, "ZDto"), "sealed class ZDto") {
			t.Fatal("opaque root/wrapper/codec dependency lost", token, err)
		}
	}
}
