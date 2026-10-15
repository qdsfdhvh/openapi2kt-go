package naming

import (
	"regexp"
	"regexp/syntax"
	"strings"

	"openapi-kt/internal/diagnostic"
)

const MaxMergeExclusionPatterns = 64
const MaxMergeExclusionPatternBytes = 1024
const MaxMergeExclusionPatternNodes = 256
const MaxMergeExclusionSuffixes = 64

// CompileMergePatterns accepts a bounded ASCII grammar shared by Go/Kotlin regex
// engines. Patterns execute only in Go against safe ASCII generated root names;
// they are never emitted into Kotlin source or used on schema instance payloads.
func CompileMergePatterns(patterns []string) ([]*regexp.Regexp, *diagnostic.Diagnostic) {
	invalid := func() *diagnostic.Diagnostic {
		return diagnostic.New("E_OPTIONS", "Merge exclusion patterns must use the bounded portable ASCII regex grammar.")
	}
	if len(patterns) > MaxMergeExclusionPatterns {
		return nil, invalid()
	}
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if len(pattern) == 0 || len(pattern) > MaxMergeExclusionPatternBytes {
			return nil, invalid()
		}
		inClass := false
		for i := 0; i < len(pattern); i++ {
			c := pattern[i]
			if c < 32 || c > 126 {
				return nil, invalid()
			}
			if c == '\\' {
				i++
				if i == len(pattern) || !strings.ContainsRune(`\.^$|?*+()[]{}`, rune(pattern[i])) {
					return nil, invalid()
				}
				continue
			}
			if c == '[' && !inClass {
				inClass = true
				continue
			}
			if c == ']' && inClass {
				inClass = false
				continue
			}
			if inClass {
				// Java class intersections and POSIX/nested classes differ from
				// Go's grammar; keep simple ranges and literal class members only.
				if c == '[' || c == '&' && i+1 < len(pattern) && pattern[i+1] == '&' {
					return nil, invalid()
				}
				continue
			}
			if c == '(' && i+1 < len(pattern) && pattern[i+1] == '?' {
				return nil, invalid()
			}
			if c == ']' || c == '}' {
				return nil, invalid()
			}
			if c == '{' {
				// Go treats malformed repetitions as literals while Java rejects
				// them. Admit only explicit decimal repetition syntax here.
				j := i + 1
				start := j
				for j < len(pattern) && asciiDigit(pattern[j]) {
					j++
				}
				if j == start {
					return nil, invalid()
				}
				if j < len(pattern) && pattern[j] == ',' {
					j++
					for j < len(pattern) && asciiDigit(pattern[j]) {
						j++
					}
				}
				if j == len(pattern) || pattern[j] != '}' {
					return nil, invalid()
				}
				i = j
			}
		}
		tree, err := syntax.Parse(pattern, syntax.Perl)
		if err != nil {
			return nil, invalid()
		}
		work := []*syntax.Regexp{tree}
		nodes := 0
		for len(work) > 0 {
			n := work[len(work)-1]
			work = work[:len(work)-1]
			nodes++
			if nodes > MaxMergeExclusionPatternNodes {
				return nil, invalid()
			}
			if n.Flags&syntax.NonGreedy != 0 {
				return nil, invalid()
			}
			if n.Op == syntax.OpRepeat && (n.Min > 64 || n.Max > 64) {
				return nil, invalid()
			}
			work = append(work, n.Sub...)
		}
		compiled, err := regexp.Compile(`\A(?:` + pattern + `)\z`)
		if err != nil {
			return nil, invalid()
		}
		result = append(result, compiled)
	}
	return result, nil
}

type MergeExclusions struct {
	exact    map[string]bool
	patterns []*regexp.Regexp
	suffixes []string
}

func (e *MergeExclusions) Matches(full string) bool {
	if e.exact[full] {
		return true
	}
	for _, pattern := range e.patterns {
		if pattern.MatchString(full) {
			return true
		}
	}
	short := full[strings.LastIndexByte(full, '.')+1:]
	for _, suffix := range e.suffixes {
		if strings.HasSuffix(short, suffix) {
			return true
		}
	}
	return false
}

func BuildMergeExclusions(enabled bool, exact, patterns, suffixes []string) (*MergeExclusions, *diagnostic.Diagnostic) {
	if failure := ValidateMergeOptions(enabled, exact); failure != nil {
		return nil, failure
	}
	if !enabled && (len(patterns) != 0 || len(suffixes) != 0) {
		return nil, diagnostic.New("E_OPTIONS", "Model merge exclusions require model merging.")
	}
	compiled, failure := CompileMergePatterns(patterns)
	if failure != nil {
		return nil, failure
	}
	if len(suffixes) > MaxMergeExclusionSuffixes {
		return nil, diagnostic.New("E_OPTIONS", "Too many merge exclusion suffixes.")
	}
	for _, suffix := range suffixes {
		valid := len(suffix) > 0 && len(suffix) <= 256
		for _, c := range []byte(suffix) {
			valid = valid && (asciiLetter(c) || asciiDigit(c) || c == '_')
		}
		if !valid {
			return nil, diagnostic.New("E_OPTIONS", "Merge exclusion suffixes must contain 1 to 256 ASCII letters, digits or underscores.")
		}
	}
	result := &MergeExclusions{exact: map[string]bool{}, patterns: compiled, suffixes: suffixes}
	for _, name := range exact {
		result.exact[name] = true
	}
	return result, nil
}

func ValidateMergeExclusions(enabled bool, exact, patterns, suffixes []string) *diagnostic.Diagnostic {
	_, failure := BuildMergeExclusions(enabled, exact, patterns, suffixes)
	return failure
}
