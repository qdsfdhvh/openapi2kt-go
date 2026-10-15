package document

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestObjectLookupAcrossIndexBoundary(t *testing.T) {
	for _, count := range []int{0, 1, 7, 8, 9, 16, 256} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			members := make([]string, count)
			for i := range members {
				members[i] = fmt.Sprintf(`"key%03d":%d`, i, i)
			}
			text := "{" + strings.Join(members, ",") + "}"
			root := mustLoad(t, text)
			check := func() {
				for i, member := range root.Members() {
					value, ok := root.Get(fmt.Sprintf("key%03d", i))
					if !ok || value != member.Value || member.Name != fmt.Sprintf("key%03d", i) {
						t.Error("lookup lost member identity or declaration order")
					}
				}
				if value, ok := root.Get("absent"); ok || value != nil {
					t.Error("missing key became present")
				}
			}
			// Lookup must remain read-only even when no hash index is present.
			var group sync.WaitGroup
			for range 4 {
				group.Go(check)
			}
			group.Wait()
			encoded, err := root.MarshalJSON()
			if err != nil || string(encoded) != text {
				t.Fatal("ordered round trip changed", err)
			}
			if count == 0 {
				return
			}
			// The read-only Members contract also has an independent lookup
			// view, used by schema provenance checks to detect accidental edits.
			original := root.Members()[0]
			root.Members()[0] = Member{Name: "changed", Value: mustLoad(t, `{}`)}
			if value, ok := root.Get(original.Name); !ok || value != original.Value {
				t.Fatal("ordered view mutation changed keyed identity")
			}
			if _, ok := root.Get("changed"); ok {
				t.Fatal("ordered view mutation changed keyed name")
			}
			root.Members()[0] = original
			for _, key := range []string{`key000`, fmt.Sprintf("key%03d", count-1), `\u006bey000`} {
				duplicate := strings.TrimSuffix(text, "}") + `,"` + key + `":null}`
				value, failure := Load([]byte(duplicate), JSON, "input.json")
				if value != nil || failure == nil || failure.Code != "E_JSON_INVALID" {
					t.Fatal("duplicate key accepted across index boundary", failure)
				}
			}
		})
	}
}
