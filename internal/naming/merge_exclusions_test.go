package naming

import (
	"strings"
	"testing"
)

func TestMergePatternPortableGrammarAndFullMatch(t *testing.T) {
	for _, pattern := range []string{`org\.example\.(Alpha|Zeta)Dto`, `.*Dto`, `[A-Z][A-Za-z0-9_.]*`, `^org\.example\.AlphaDto$`, `A{1,64}`, `[?+]`, `a\+?`, `a\??`, `\{1\}`, `\]`} {
		if _, err := CompileMergePatterns([]string{pattern}); err != nil {
			t.Fatal(pattern, err)
		}
	}
	for _, pattern := range []string{"", `(?=Alpha)`, `(?<=Alpha)`, `(Alpha)\1`, `(?i)alpha`, `(?:Alpha)`, `\p{L}`, `\d+`, `\QAlpha\E`, `Alpha+?`, `Alpha++`, `[z-a]`, `[[:alpha:]]`, `[a-z&&[^b]]`, `A{1,65}`, `A{bad}`, `A{`, `A}`, `A]`, "é", "A\n", strings.Repeat("A", 1025)} {
		if _, err := CompileMergePatterns([]string{pattern}); err == nil {
			t.Fatal("unsupported regex accepted", pattern)
		}
	}
	patterns := make([]string, 65)
	for i := range patterns {
		patterns[i] = "A"
	}
	if _, err := CompileMergePatterns(patterns); err == nil {
		t.Fatal("unbounded pattern inventory")
	}
	exclusions, err := BuildMergeExclusions(true, []string{"org.example.ExactDto"}, []string{`org\.example\.(Alpha|Zeta)Dto`}, []string{"Dto2", "_"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"org.example.ExactDto", "org.example.AlphaDto", "org.example.ZetaDto", "org.other.RecordDto2", "org.other.Record_"} {
		if !exclusions.Matches(name) {
			t.Fatal(name)
		}
	}
	for _, name := range []string{"org.other.AlphaDto", "org.example.AlphaDto.Other", "xorg.example.AlphaDto", "org.example.AlphaDto2x", "org.other.recorddto2"} {
		if exclusions.Matches(name) {
			t.Fatal(name)
		}
	}
	for _, suffix := range []string{"", ".Dto", "é", "*", strings.Repeat("D", 257)} {
		if err := ValidateMergeExclusions(true, nil, nil, []string{suffix}); err == nil {
			t.Fatal(suffix)
		}
	}
	if err := ValidateMergeExclusions(false, nil, []string{".*"}, nil); err == nil {
		t.Fatal("patterns without merge")
	}
	if err := ValidateMergeExclusions(false, nil, nil, []string{"Dto"}); err == nil {
		t.Fatal("suffix without merge")
	}
	// Same cases run through Kotlin Regex.matches in the fresh consumer.
	for _, tc := range []struct {
		pattern string
		want    bool
	}{
		{`org\.example\.exclusions\.models\.(Alpha|Zeta)Dto`, true},
		{`AlphaDto`, false}, {`[A-Za-z0-9_.]+`, true},
		{`^org\.example\.exclusions\.models\.AlphaDto$`, true},
		{`.*\.AlphaDto{1,2}`, true}, {`org.*ZetaDto`, false},
	} {
		e, failure := BuildMergeExclusions(true, nil, []string{tc.pattern}, nil)
		if failure != nil || e.Matches("org.example.exclusions.models.AlphaDto") != tc.want {
			t.Fatal(tc, failure)
		}
	}
}
