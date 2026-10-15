package schema

import (
	"encoding/json"
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
	"sort"
	"strings"
)

const MaxNameLockBytes = 1 << 20
const MaxNameLockEntries = 4096
const MaxNameLockOccurrences = 16384

type NameLockProfile struct {
	Package   string `json:"package"`
	Suffix    string `json:"suffix"`
	Qualified bool   `json:"qualified"`
}
type NameLockEntry struct {
	Role        string   `json:"role"`
	Shape       string   `json:"shape"`
	Name        string   `json:"name"`
	Occurrences []string `json:"occurrences"`
}
type HoistedNameLock struct {
	Version int             `json:"version"`
	Profile NameLockProfile `json:"profile"`
	Entries []NameLockEntry `json:"entries"`
}

func nameLockError(code, message string) *diagnostic.Diagnostic { return diagnostic.New(code, message) }
func lockIdentifier(s string) bool {
	if len(s) == 0 || s[0] < 'A' || s[0] > 'Z' || naming.ReservedPathSegment(s) {
		return false
	}
	for _, c := range []byte(s) {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}
func lockKeys(v *document.Value, keys ...string) bool {
	if v == nil || v.Kind() != document.Object || len(v.Members()) != len(keys) {
		return false
	}
	for _, key := range keys {
		if !has(v, key) {
			return false
		}
	}
	return true
}
func ParseNameLock(data []byte, pkg, suffix string, qualified bool) (*HoistedNameLock, *diagnostic.Diagnostic) {
	wanted := NameLockProfile{pkg, suffix, qualified}
	fresh := &HoistedNameLock{Version: 1, Profile: wanted, Entries: []NameLockEntry{}}
	if data == nil {
		return fresh, nil
	}
	bad := func() (*HoistedNameLock, *diagnostic.Diagnostic) {
		return nil, nameLockError("E_NAME_LOCK_INVALID", "Hoisted name lock is malformed; no files were generated.")
	}
	if len(data) > MaxNameLockBytes {
		return nil, nameLockError("E_NAME_LOCK_LIMIT", "Hoisted name lock exceeds metadata limits.")
	}
	v, e := document.ParseObject(data, MaxNameLockBytes)
	if e != nil || !lockKeys(v, "version", "profile", "entries") {
		return bad()
	}
	token, ok := get(v, "version").Numeric()
	if !ok {
		return bad()
	}
	if string(token) != "1" {
		return nil, nameLockError("E_NAME_LOCK_VERSION", "Hoisted name lock version is unsupported.")
	}
	profile := get(v, "profile")
	if !lockKeys(profile, "package", "suffix", "qualified") {
		return bad()
	}
	p, a := get(profile, "package").Text()
	s, b := get(profile, "suffix").Text()
	q, c := get(profile, "qualified").Bool()
	if !a || !b || !c || naming.ValidateOptions(p, s) != nil {
		return bad()
	}
	if (NameLockProfile{p, s, q}) != wanted {
		return nil, nameLockError("E_NAME_LOCK_PROFILE", "Hoisted name lock profile differs from generation options.")
	}
	entries := get(v, "entries")
	if entries.Kind() != document.Array {
		return bad()
	}
	if len(entries.Items()) > MaxNameLockEntries {
		return nil, nameLockError("E_NAME_LOCK_LIMIT", "Hoisted name lock exceeds metadata limits.")
	}
	names, origins := map[string]bool{}, map[string]bool{}
	last := ""
	total := 0
	for _, item := range entries.Items() {
		if !lockKeys(item, "role", "shape", "name", "occurrences") {
			return bad()
		}
		role, a := get(item, "role").Text()
		shape, b := get(item, "shape").Text()
		name, c := get(item, "name").Text()
		if !a || !b || !c || !lockIdentifier(role) || !lockIdentifier(name) || len(shape) != 64 {
			return bad()
		}
		for _, ch := range []byte(shape) {
			if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
				return bad()
			}
		}
		key := role + ":" + shape
		if key <= last || names[strings.ToLower(name)] {
			return bad()
		}
		last = key
		names[strings.ToLower(name)] = true
		raw := get(item, "occurrences")
		if raw.Kind() != document.Array || len(raw.Items()) == 0 {
			return bad()
		}
		entry := NameLockEntry{role, shape, name, []string{}}
		prior := ""
		for _, value := range raw.Items() {
			origin, valid := value.Text()
			if !valid || origin <= prior || !strings.HasPrefix(origin, "/") || strings.ContainsAny(origin, "\x00\r\n") || origins[role+":"+origin] {
				return bad()
			}
			origins[role+":"+origin] = true
			prior = origin
			entry.Occurrences = append(entry.Occurrences, origin)
			total++
			if total > MaxNameLockOccurrences {
				return nil, nameLockError("E_NAME_LOCK_LIMIT", "Hoisted name lock exceeds metadata limits.")
			}
		}
		fresh.Entries = append(fresh.Entries, entry)
	}
	return fresh, nil
}
func (state *HoistedNameLock) Encode() ([]byte, *diagnostic.Diagnostic) {
	sort.Slice(state.Entries, func(i, j int) bool {
		return state.Entries[i].Role+":"+state.Entries[i].Shape < state.Entries[j].Role+":"+state.Entries[j].Shape
	})
	data, e := json.Marshal(state)
	if len(data)+1 > MaxNameLockBytes {
		return nil, nameLockError("E_NAME_LOCK_LIMIT", "Hoisted name lock exceeds metadata limits.")
	}
	if e != nil {
		panic("invalid internal name lock")
	}
	if _, err := ParseNameLock(data, state.Profile.Package, state.Profile.Suffix, state.Profile.Qualified); err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
