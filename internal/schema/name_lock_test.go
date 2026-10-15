package schema

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNameLockV1GoldenSemanticFingerprint(t *testing.T) {
	one := int64(1)
	d := &Model{ClassName: "AlphaResponse", Properties: []Property{{Name: "id", WireName: "id", Type: TypeRef{Kind: Scalar, Name: "kotlin.String", SchemaSteps: 30}, Validation: &PrimitiveValidation{MinLength: &one}}}}
	const golden = "974d05908ca2c53d5b21fac95f4b93443868e35358276f100c168ef6b796d10c"
	if nameLockShapeV1(d, "org.example") != golden {
		t.Fatal("v1 protocol drift", nameLockShapeV1(d, "org.example"))
	}
	d.ClassName = "BetaResponse"
	d.Plain = true
	d.Properties[0].Type.SchemaSteps = 48
	if nameLockShapeV1(d, "org.example") != golden {
		t.Fatal("presentation/work leaked into stable protocol")
	}
	d.Properties[0].Type.Serializer = "org.example.ProviderCodec"
	if nameLockShapeV1(d, "org.example") == golden {
		t.Fatal("opaque codec erased")
	}
}
func TestNameLockStrictEntriesAndBudgets(t *testing.T) {
	base := HoistedNameLock{Version: 1, Profile: NameLockProfile{"org.example", "", false}, Entries: []NameLockEntry{{Role: "Response", Shape: strings.Repeat("0", 64), Name: "Response", Occurrences: []string{"/a", "/b"}}}}
	for _, change := range []string{"namecase", "order", "empty", "duporigin", "hash", "unknown", "profilenull"} {
		data, _ := json.Marshal(base)
		s := string(data)
		switch change {
		case "namecase":
			s = strings.Replace(s, `"name":"Response"`, `"name":"when"`, 1)
		case "order":
			s = strings.Replace(s, `["/a","/b"]`, `["/b","/a"]`, 1)
		case "empty":
			s = strings.Replace(s, `["/a","/b"]`, `[]`, 1)
		case "duporigin":
			s = strings.Replace(s, `["/a","/b"]`, `["/a","/a"]`, 1)
		case "hash":
			s = strings.Replace(s, strings.Repeat("0", 64), strings.Repeat("G", 64), 1)
		case "unknown":
			s = strings.Replace(s, `"name":"Response"`, `"name":"Response","extra":false`, 1)
		case "profilenull":
			s = strings.Replace(s, `"qualified":false`, `"qualified":null`, 1)
		}
		if _, e := ParseNameLock([]byte(s), "org.example", "", false); e == nil || e.Code != "E_NAME_LOCK_INVALID" {
			t.Fatal(change, e)
		}
	}
	// Different roles may share a source pointer; within one role never duplicate.
	for _, count := range []int{MaxNameLockEntries, MaxNameLockEntries + 1} {
		state := base
		state.Entries = []NameLockEntry{}
		for i := 0; i < count; i++ {
			state.Entries = append(state.Entries, NameLockEntry{Role: "Response", Shape: fmt.Sprintf("%064x", i), Name: fmt.Sprintf("Response%d", i), Occurrences: []string{fmt.Sprintf("/o%d", i)}})
		}
		data, _ := json.Marshal(state)
		_, e := ParseNameLock(data, "org.example", "", false)
		if count == MaxNameLockEntries && e != nil || count > MaxNameLockEntries && (e == nil || e.Code != "E_NAME_LOCK_LIMIT") {
			t.Fatal(count, e)
		}
	}
	for _, count := range []int{MaxNameLockOccurrences, MaxNameLockOccurrences + 1} {
		state := base
		state.Entries[0].Occurrences = []string{}
		for i := 0; i < count; i++ {
			state.Entries[0].Occurrences = append(state.Entries[0].Occurrences, fmt.Sprintf("/o%05d", i))
		}
		data, _ := json.Marshal(state)
		_, e := ParseNameLock(data, "org.example", "", false)
		if count == MaxNameLockOccurrences && e != nil || count > MaxNameLockOccurrences && (e == nil || e.Code != "E_NAME_LOCK_LIMIT") {
			t.Fatal(count, e)
		}
	}
}
func FuzzNameLockV1(f *testing.F) {
	for _, s := range []string{"{}", `{"version":2,"profile":{},"entries":[]}`, `{"version":1,"profile":{"package":"org.example","suffix":"","qualified":false},"entries":[]}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		state, e := ParseNameLock([]byte(s), "org.example", "", false)
		if e == nil {
			data, e := state.Encode()
			if e != nil {
				t.Fatal(e)
			}
			if _, e := ParseNameLock(data, "org.example", "", false); e != nil {
				t.Fatal(e)
			}
		}
	})
}
