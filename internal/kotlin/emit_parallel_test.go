package kotlin

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"

	"openapi-kt/internal/document"
	"openapi-kt/internal/schema"
	"openapi-kt/internal/testutil"
)

func TestParallelEmissionCorpus(t *testing.T) {
	for _, c := range testutil.Corpus(t) {
		if c.Expect.Kind != "success" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			root, failure := document.Load(testutil.Read(t, c.Input), document.JSON, c.Input)
			if failure != nil {
				t.Fatal(failure)
			}
			plan, _, failure := schema.Analyze(root, schema.Options{PackageName: c.PackageName, Suffix: c.ModelNameSuffix})
			if failure != nil {
				t.Fatal(failure)
			}
			// Reuse the very same declaration pointers across worker slots. This
			// exercises shared nested plans and would expose lazy emitter writes.
			declarations := slices.Repeat(plan.Models, 67)
			before, err := json.Marshal(declarations)
			if err != nil {
				t.Fatal(err)
			}
			want := emitFiles(declarations, c.PackageName, plan.EmissionHints, 1)
			for _, workers := range []int{2, 4, 8} {
				if got := emitFiles(declarations, c.PackageName, plan.EmissionHints, workers); !reflect.DeepEqual(got, want) {
					t.Fatalf("%d workers changed file order or content", workers)
				}
			}
			if got := Emit(declarations, c.PackageName, plan.EmissionHints); !reflect.DeepEqual(got, want) {
				t.Fatal("automatic scheduling changed file order or content")
			}
			after, err := json.Marshal(declarations)
			if err != nil || string(after) != string(before) {
				t.Fatal("emission mutated the plan", err)
			}
		})
	}
}

func TestParallelEmissionSharedHints(t *testing.T) {
	minimum := int64(1)
	checks := &schema.PrimitiveValidation{MinLength: &minimum}
	model := &schema.Model{ClassName: "Record", Properties: []schema.Property{{Name: "value", WireName: "value", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String", Validation: checks}}}}
	hints := schema.EmissionHints{checks: true}
	declarations := slices.Repeat([]schema.Declaration{model}, 129)
	want := emitFiles(declarations, "org.example.parallel", hints, 1)
	var group sync.WaitGroup
	for range 4 {
		group.Go(func() {
			if got := Emit(declarations, "org.example.parallel", hints); !reflect.DeepEqual(got, want) {
				t.Error("concurrent emission changed hinted output")
			}
		})
	}
	group.Wait()
	if len(hints) != 1 || !hints[checks] {
		t.Fatal("emission changed hints")
	}
}

func TestEmissionBatchBoundaries(t *testing.T) {
	for _, count := range []int{0, 1, 31, 32, 63, 64, 65, 255, 256, 257} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			declarations := benchmarkDeclarations(count)
			want := emitFiles(declarations, "org.example.parallel", nil, 1)
			if got := Emit(declarations, "org.example.parallel", nil); !reflect.DeepEqual(got, want) {
				t.Fatal("missing, reordered or changed files at scheduling boundary")
			}
		})
	}
}

// Deterministic workload: 16 scalar/list/reference fields and one nested DTO
// per model, no random source or external files. Parsing/analysis are excluded.
func benchmarkDeclarations(count int) []schema.Declaration {
	declarations := make([]schema.Declaration, count)
	for i := range declarations {
		model := &schema.Model{ClassName: fmt.Sprintf("Model%05d", i)}
		for j := range 16 {
			ty := schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String"}
			switch j % 4 {
			case 1:
				ty = schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.Long"}
			case 2:
				element := ty.Null()
				ty = schema.TypeRef{Kind: schema.List, Element: &element}
			case 3:
				ty = schema.TypeRef{Kind: schema.Named, Name: fmt.Sprintf("org.example.benchmark.models.Model%05d", (i+1)%count)}
			}
			if j%2 == 0 {
				ty = ty.Null()
			}
			name := fmt.Sprintf("field%02d", j)
			model.Properties = append(model.Properties, schema.Property{Name: name, WireName: name, Type: ty, DefaultNull: j%2 == 0})
		}
		model.Nested = []schema.Declaration{&schema.Model{ClassName: "Nested", Properties: []schema.Property{{Name: "value", WireName: "value", Type: schema.TypeRef{Kind: schema.Scalar, Name: "kotlin.String"}}}}}
		declarations[i] = model
	}
	return declarations
}

func BenchmarkEmit(b *testing.B) {
	for _, count := range []int{100, 1000, 4000} {
		declarations := benchmarkDeclarations(count)
		want := emitFiles(declarations, "org.example.benchmark", nil, 1)
		for _, workers := range []int{1, 2, 4, 8} {
			b.Run(fmt.Sprintf("models=%d/workers=%d", count, workers), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					got := emitFiles(declarations, "org.example.benchmark", nil, workers)
					b.StopTimer()
					if !reflect.DeepEqual(got, want) {
						b.Fatal("emission changed artifacts")
					}
					b.StartTimer()
				}
			})
		}
	}
}
