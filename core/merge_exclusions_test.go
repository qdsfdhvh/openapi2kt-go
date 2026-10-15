package core_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"openapi-kt/core"
)

func TestMergeRegexAndSuffixExclusionsProtectCanonicalAndLosingRoots(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/http/merge-models.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []core.ModelStyle{core.DataClass, core.PlainClass} {
		for _, kind := range []string{"regexWinner", "regexLoser", "suffixWinner", "suffixLoser", "mixed", "unknown"} {
			t.Run(string(style)+kind, func(t *testing.T) {
				r := request(input, "Dto")
				r.Options.PackageName = "org.example.merge"
				r.Options.ModelStyle = style
				r.Options.MergeIdenticalModels = true
				r.Options.Validation = true
				keep := "AlphaDto"
				switch kind {
				case "regexWinner":
					r.Options.ExcludeMergeModelPatterns = []string{`org\.example\.merge\.models\.AlphaDto`}
				case "regexLoser":
					r.Options.ExcludeMergeModelPatterns = []string{`org\.example\.merge\.models\.(Zeta|ChildZ)Dto`}
					keep = "ZetaDto"
				case "suffixWinner":
					r.Options.ExcludeMergeModelSuffixes = []string{"AlphaDto"}
				case "suffixLoser":
					r.Options.ExcludeMergeModelSuffixes = []string{"ZetaDto"}
					keep = "ZetaDto"
				case "mixed":
					r.Options.ExcludeMergeModelPatterns = []string{`.*\.AlphaDto`}
					r.Options.ExcludeMergeModelSuffixes = []string{"ChildZDto"}
				case "unknown":
					r.Options.ExcludeMergeModelPatterns = []string{`AlphaDto`}
					r.Options.ExcludeMergeModelSuffixes = []string{"alphadto"}
					keep = ""
				}
				got, err := core.GenerateWithDiagnostics(r)
				if err != nil {
					t.Fatal(err)
				}
				if keep != "" {
					for _, name := range []string{"AlphaDto", "ZetaDto"} {
						if strings.Contains(source(t, got, name), "typealias "+name) {
							t.Fatal("protected winner/loser absorbed its peer", kind, name)
						}
					}
					if kind == "regexLoser" || kind == "mixed" {
						if strings.Contains(source(t, got, "ParentZDto"), "typealias ParentZDto") {
							t.Fatal("excluded child lost dependent nominal identity")
						}
					}
				} else if !strings.Contains(source(t, got, "ZetaDto"), "typealias ZetaDto") {
					t.Fatal("regex used substring or suffix case-folding")
				}
				repeat, e := core.GenerateWithDiagnostics(r)
				if e != nil || !reflect.DeepEqual(got, repeat) {
					t.Fatal(e)
				}
			})
		}
	}
}
