package output

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRootMoveDestinationFirstEveryBoundaryRollback(t *testing.T) {
	src, dst, s := moveFixture(t, 3)
	nextSource := filepath.Join(filepath.Dir(src), "z-source")
	nextTarget := filepath.Join(filepath.Dir(src), "a-target")
	if e := os.Rename(src, nextSource); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(dst, nextTarget); e != nil {
		t.Fatal(e)
	}
	src, dst = nextSource, nextTarget
	current, err := ReadRootMoveState(src)
	if err != nil {
		t.Fatal(err)
	}
	s = current
	a, b := snapshot(t, src), snapshot(t, dst)
	for ri := 0; ri < 2; ri++ {
		for op := 0; op < len(s.Entries); op++ {
			_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, operation int, _ string) error {
				if index == ri && phase == "apply" && operation == op {
					return errors.New("destination first fault")
				}
				return nil
			})
			errorCode(t, e, "E_OUTPUT_COMMIT")
			if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
				t.Fatal("destination-first rollback failed", ri, op)
			}
		}
	}
}

func FuzzRootMoveRollback(f *testing.F) {
	for _, seed := range []uint8{0, 1, 2, 3, 4, 7, 11, 19, 255} {
		f.Add(seed, seed%4, seed%2 == 0)
	}
	f.Fuzz(func(t *testing.T, boundary, version uint8, reverse bool) {
		src, dst, s := moveFixture(t, int(version%4)+1)
		if reverse {
			nextSource, nextTarget := filepath.Join(filepath.Dir(src), "z-source"), filepath.Join(filepath.Dir(src), "a-target")
			if e := os.Rename(src, nextSource); e != nil {
				t.Fatal(e)
			}
			if e := os.Rename(dst, nextTarget); e != nil {
				t.Fatal(e)
			}
			src, dst = nextSource, nextTarget
			current, err := ReadRootMoveState(src)
			if err != nil {
				t.Fatal(err)
			}
			s = current
		}
		a, b := snapshot(t, src), snapshot(t, dst)
		n := len(s.Entries)
		at := int(boundary) % (2 * n)
		_, e := moveOutputRoot(src, dst, s, false, func(index int, phase string, op int, _ string) error {
			if phase == "apply" && index*n+op == at {
				return errors.New("bounded move fault")
			}
			return nil
		})
		errorCode(t, e, "E_OUTPUT_COMMIT")
		if !reflect.DeepEqual(a, snapshot(t, src)) || !reflect.DeepEqual(b, snapshot(t, dst)) {
			t.Fatal("model: failed move must preserve both trees")
		}
		if changed, e := MoveOutputRoot(src, dst, s, false); e != nil || !changed {
			t.Fatal("retry failed", changed, e)
		}
		next, e := ReadRootMoveState(dst)
		expected := s
		expected.root = next.root
		if e != nil || !reflect.DeepEqual(next, expected) {
			t.Fatal("model: success must preserve complete state", e)
		}
	})
}
