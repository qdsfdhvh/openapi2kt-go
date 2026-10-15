package core_test

import (
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestMergeExclusionsOriginalOptionPrecedenceAndFailureReport(t *testing.T) {
	for _, kind := range []string{"disabled", "lookaround", "backreference", "patternSize", "patternCount", "suffixCount", "suffix", "repeatBound", "syntaxNodes"} {
		t.Run(kind, func(t *testing.T) {
			r := request([]byte("malformed input must not be parsed first"), "Dto")
			r.Options.MergeIdenticalModels = true
			switch kind {
			case "disabled":
				r.Options.MergeIdenticalModels = false
				r.Options.ExcludeMergeModelSuffixes = []string{"Dto"}
			case "lookaround":
				r.Options.ExcludeMergeModelPatterns = []string{"(?=x)"}
			case "backreference":
				r.Options.ExcludeMergeModelPatterns = []string{`(x)\1`}
			case "patternSize":
				r.Options.ExcludeMergeModelPatterns = []string{strings.Repeat("x", 1025)}
			case "patternCount":
				r.Options.ExcludeMergeModelPatterns = make([]string, 65)
				for i := range r.Options.ExcludeMergeModelPatterns {
					r.Options.ExcludeMergeModelPatterns[i] = ".*"
				}
			case "suffixCount":
				r.Options.ExcludeMergeModelSuffixes = make([]string, 65)
				for i := range r.Options.ExcludeMergeModelSuffixes {
					r.Options.ExcludeMergeModelSuffixes[i] = "Dto"
				}
			case "suffix":
				r.Options.ExcludeMergeModelSuffixes = []string{".Dto"}
			case "repeatBound":
				r.Options.ExcludeMergeModelPatterns = []string{"x{1,65}"}
			case "syntaxNodes":
				r.Options.ExcludeMergeModelPatterns = []string{strings.Repeat("(a*)", 129)}
			}
			got, err := core.GenerateWithDiagnostics(r)
			if err == nil || err.(core.Diagnostics)[0].Code != "E_OPTIONS" || !reflect.DeepEqual(got, core.GenerationReport{}) {
				t.Fatal(kind, err, got)
			}
		})
	}
}

func TestMergeExclusionBoundaryInventoriesDoNotChangeUnmatchedArtifacts(t *testing.T) {
	r := request([]byte(`{"openapi":"3.1.0","components":{"schemas":{"Record":{"properties":{"id":{"type":"integer"}}}}}}`), "Dto")
	r.Options.MergeIdenticalModels = true
	control, err := core.GenerateWithDiagnostics(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Options.ExcludeMergeModelPatterns = make([]string, 64)
	r.Options.ExcludeMergeModelSuffixes = make([]string, 64)
	for i := range r.Options.ExcludeMergeModelPatterns {
		r.Options.ExcludeMergeModelPatterns[i] = strings.Repeat("x", 1024)
		r.Options.ExcludeMergeModelSuffixes[i] = strings.Repeat("Z", 256)
	}
	got, err := core.GenerateWithDiagnostics(r)
	if err != nil || !reflect.DeepEqual(control, got) {
		t.Fatal(err)
	}
}
