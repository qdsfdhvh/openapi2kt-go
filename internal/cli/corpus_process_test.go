package cli_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"openapi-kt/internal/testutil"
)

func TestOriginalCorpusThroughActualGoBinary(t *testing.T) {
	positives, negatives, snapshots, graduated := 0, 0, 0, 0
	for _, c := range testutil.Corpus(t) {
		t.Run(c.ID, func(t *testing.T) {
			p := newProject(t)
			result(t, p.generate(t), 0)
			before := snapshot(t, p.Output)
			name := filepath.Base(c.Input)
			if c.SourceName != nil {
				name = *c.SourceName
			}
			p.Input = filepath.Join(p.Root, name)
			put(t, p.Input, testutil.Read(t, c.Input))
			args := []string{"generate", "--input", p.Input, "--output", p.Output, "--package-name", c.PackageName, "--model-name-suffix", c.ModelNameSuffix, "--input-format", c.InputFormat, "--diagnostic-format", "json"}
			if testutil.ScalarGraduation(c) || testutil.ArrayGraduation(c) || testutil.NullableRawGraduation(c) || testutil.BooleanValueGraduation(c) || testutil.DefaultGraduation(c) {
				graduated++
				generated := result(t, invoke(t, "", args...), 0)
				files := map[string]string{}
				for _, path := range generated.Files {
					files[path] = string(read(t, filepath.Join(p.Output, filepath.FromSlash(path))))
				}
				if testutil.DefaultGraduation(c) {
					testutil.CheckDefaultGraduation(t, c, files)
				} else if testutil.BooleanValueGraduation(c) {
					testutil.CheckBooleanValueGraduation(t, c, files, generated.Warnings)
				} else if testutil.NullableRawGraduation(c) {
					testutil.CheckNullableRawGraduation(t, c, files)
					if len(generated.Warnings) != 1 || generated.Warnings[0].Code != "W_JSON_ELEMENT_APPROXIMATION" || *generated.Warnings[0].Pointer != "/components/schemas/Record" {
						t.Fatal(generated.Warnings)
					}
				} else if testutil.ArrayGraduation(c) {
					testutil.CheckArrayGraduation(t, c, files)
				} else {
					testutil.CheckScalarGraduation(t, c, files)
				}
				current := snapshot(t, p.Output)
				result(t, invoke(t, "", append(args, "--check")...), 0)
				result(t, invoke(t, "", args...), 0)
				if !reflect.DeepEqual(snapshot(t, p.Output), current) {
					t.Fatal("graduation no-op touched files")
				}
				result(t, invoke(t, "", "validate", "--input", p.Input, "--package-name", c.PackageName, "--input-format", c.InputFormat, "--diagnostic-format", "json"), 0)
				return
			}
			if c.Expect.Kind == "error" {
				negatives++
				for _, check := range []bool{true, false} {
					invocationArgs := append([]string(nil), args...)
					if check {
						invocationArgs = append(invocationArgs, "--check")
					}
					report := fails(t, invoke(t, "", invocationArgs...), 1, c.Expect.Code)
					if !reflect.DeepEqual(report.Diagnostics[0].Pointer, c.Expect.Pointer) {
						t.Fatalf("pointer want %v got %+v", c.Expect.Pointer, report.Diagnostics[0])
					}
					if !reflect.DeepEqual(snapshot(t, p.Output), before) {
						t.Fatal("negative modified prior tree/bytes/mtimes")
					}
				}
			} else {
				positives++
				generated := result(t, invoke(t, "", args...), 0)
				want := []string{}
				for path := range c.Expect.Files {
					want = append(want, path)
				}
				sort.Strings(want)
				if !reflect.DeepEqual(generated.Files, want) || len(generated.Diagnostics) != 0 {
					t.Fatal("file inventory drift", generated, want)
				}
				for path, golden := range c.Expect.Files {
					snapshots++
					if !bytes.Equal(read(t, filepath.Join(p.Output, filepath.FromSlash(path))), testutil.CurrentUUIDSnapshot(t, c, path, testutil.Read(t, golden))) {
						t.Fatal("original golden drift", path)
					}
				}
				current := snapshot(t, p.Output)
				result(t, invoke(t, "", append(args, "--check")...), 0)
				result(t, invoke(t, "", args...), 0)
				if !reflect.DeepEqual(snapshot(t, p.Output), current) {
					t.Fatal("corpus check/no-op touched files")
				}
			}
			if c.Validate {
				validation := invoke(t, "", "validate", "--input", p.Input, "--package-name", c.PackageName, "--input-format", c.InputFormat, "--diagnostic-format", "json")
				if c.Expect.Kind == "error" {
					report := fails(t, validation, 1, c.Expect.Code)
					if !reflect.DeepEqual(report.Diagnostics[0].Pointer, c.Expect.Pointer) {
						t.Fatal("validate pointer drift", report)
					}
				} else {
					result(t, validation, 0)
				}
			}
		})
	}
	if positives != 42 || negatives != 99 || snapshots != 54 || graduated != 13 {
		t.Fatal("corpus inventory drift", positives, negatives, snapshots, graduated)
	}
	t.Logf("Actual native Go command: %d current positive (including two YAML acceptances) + %d remaining negative + %d explicit scalar/array/nullable-raw/boolean-value/root/default graduations, %d reviewed snapshots", positives, negatives, graduated, snapshots)
}
