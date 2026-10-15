package naming

import "testing"

func FuzzMergeExclusionPattern(f *testing.F) {
	for _, seed := range []string{`.*Dto`, `org\.example\.(Alpha|Zeta)Dto`, `(?=bad)`, `(x)\1`, `[A-Z]`, `[a-z&&[^b]]`, `x{1,64}`, `x{1,65}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, pattern string) {
		if len(pattern) > 2048 {
			return
		}
		a, e := BuildMergeExclusions(true, nil, []string{pattern}, nil)
		b, d := BuildMergeExclusions(true, nil, []string{pattern}, nil)
		if (e == nil) != (d == nil) {
			t.Fatal("nondeterministic grammar")
		}
		if e != nil {
			return
		}
		for _, name := range []string{"org.example.models.RecordDto", "org.example.api.models.FetchErrorDto", "org.example.models.AlphaDto"} {
			if a.Matches(name) != b.Matches(name) {
				t.Fatal("nondeterministic matcher")
			}
		}
	})
}
