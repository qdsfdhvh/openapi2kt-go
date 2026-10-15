package cli_test

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDTOPropertyCountFixture(t *testing.T) {
	p := newProject(t)
	input := read(t, filepath.Join(repo, "tests", "fixtures", "http", "dto-property-counts.json"))
	put(t, p.Input, input)
	result(t, p.generate(t, "--validation"), 0)
	before := snapshot(t, p.Output)
	result(t, p.generate(t, "--validation", "--check"), 0)
	result(t, p.generate(t, "--validation"), 0)
	if !reflect.DeepEqual(before, snapshot(t, p.Output)) {
		t.Fatal("non-deterministic/noop")
	}
}
