package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"openapi-kt/core"
	"openapi-kt/internal/output"
	"openapi-kt/internal/schema"
)

// Allocation inventories are deliberately complete, including groups with no
// surviving occurrence match. Changes correlate exact role+source pointers;
// neither a moved pointer nor a split/merge is guessed to be the same API.
type nameLockSymbol struct {
	Role        string   `json:"role"`
	Shape       string   `json:"shape"`
	Classifier  string   `json:"classifier"`
	Descriptor  string   `json:"descriptor"`
	Occurrences []string `json:"occurrences"`
}
type nameLockChange struct {
	Role             string `json:"role"`
	Occurrence       string `json:"occurrence"`
	Before           string `json:"before,omitempty"`
	After            string `json:"after,omitempty"`
	DescriptorBefore string `json:"descriptor_before,omitempty"`
	DescriptorAfter  string `json:"descriptor_after,omitempty"`
	ShapeChanged     bool   `json:"shape_changed"`
}
type nameLockFileChange struct {
	Path   string `json:"path"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}
type nameLockMigration struct {
	Action  output.NameLockAction `json:"action"`
	Spec    string                `json:"spec,omitempty"`
	Before  []nameLockSymbol      `json:"before"`
	After   []nameLockSymbol      `json:"after"`
	Changes []nameLockChange      `json:"changes"`
	Files   []nameLockFileChange  `json:"files"`
	Impact  string                `json:"impact"`
}

// The current emitter spells top-level aliases with fully qualified targets.
// Follow those aliases so a post-hoist merge's descriptor is reported as the
// canonical ordinary DTO/enum, rather than the pre-merge lock allocation.
func nameLockSymbols(state *schema.HoistedNameLock, files []core.GeneratedFile) []nameLockSymbol {
	aliases := map[string]string{}
	for _, file := range files {
		pkg := ""
		for line := range strings.SplitSeq(file.Content, "\n") {
			if strings.HasPrefix(line, "package ") {
				pkg = strings.TrimPrefix(line, "package ")
			}
			if strings.HasPrefix(line, "public typealias ") {
				name, target, ok := strings.Cut(strings.TrimPrefix(line, "public typealias "), " = ")
				if ok {
					aliases[pkg+"."+name] = target
				}
			}
		}
	}
	out := make([]nameLockSymbol, 0, len(state.Entries))
	for _, entry := range state.Entries {
		classifier := state.Profile.Package + ".models." + entry.Name
		descriptor := classifier
		seen := map[string]bool{}
		for aliases[descriptor] != "" && !seen[descriptor] {
			seen[descriptor] = true
			descriptor = aliases[descriptor]
		}
		out = append(out, nameLockSymbol{entry.Role, entry.Shape, classifier, descriptor, entry.Occurrences})
	}
	return out
}
func symbolOccurrences(symbols []nameLockSymbol) map[string]nameLockSymbol {
	out := map[string]nameLockSymbol{}
	for _, symbol := range symbols {
		for _, occurrence := range symbol.Occurrences {
			out[symbol.Role+":"+occurrence] = symbol
		}
	}
	return out
}
func fileHashes(files []core.GeneratedFile) map[string]string {
	out := map[string]string{}
	for _, file := range files {
		hash := sha256.Sum256([]byte(file.Content))
		out[file.Path] = hex.EncodeToString(hash[:])
	}
	return out
}
func unionKeys[V any](before, after map[string]V) []string {
	keys := map[string]bool{}
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
func nameLockMigrationReport(args arguments, owner string, previous []byte, artifacts core.GenerationArtifacts) (nameLockMigration, *core.Diagnostic) {
	old, e := schema.ParseNameLock(previous, args.packageName, args.suffix, args.qualifyHoistedNames)
	if e != nil {
		return nameLockMigration{}, e
	}
	next, e := schema.ParseNameLock(artifacts.NameLock, args.packageName, args.suffix, args.qualifyHoistedNames)
	if e != nil {
		return nameLockMigration{}, e
	}
	files, e := output.ReadNameLockFiles(args.output, owner, previous)
	if e != nil {
		return nameLockMigration{}, e
	}
	result := nameLockMigration{
		Action: args.nameLockAction, Spec: owner,
		Before: nameLockSymbols(old, files), After: nameLockSymbols(next, artifacts.Files),
		Changes: []nameLockChange{}, Files: []nameLockFileChange{},
		Impact: "Review shared-model imports and nested classifier references, API root aliases, JVM class and serializer descriptor identities; rebuild compiled consumers. Shape changes require separate constructor/validation/wire review. This report correlates exact role and source pointers, not moved APIs or wire equivalence.",
	}
	before, after := symbolOccurrences(result.Before), symbolOccurrences(result.After)
	for _, key := range unionKeys(before, after) {
		b, a := before[key], after[key]
		if b.Classifier == a.Classifier && b.Descriptor == a.Descriptor && b.Shape == a.Shape {
			continue
		}
		role, occurrence, _ := strings.Cut(key, ":")
		result.Changes = append(result.Changes, nameLockChange{role, occurrence, b.Classifier, a.Classifier, b.Descriptor, a.Descriptor, b.Shape != a.Shape})
	}
	oldHashes, newHashes := fileHashes(files), fileHashes(artifacts.Files)
	for _, path := range unionKeys(oldHashes, newHashes) {
		if oldHashes[path] != newHashes[path] {
			result.Files = append(result.Files, nameLockFileChange{path, oldHashes[path], newHashes[path]})
		}
	}
	if owner != "" {
		result.Impact += " File differences describe this selected owner's membership; shared paths may remain owned by peers."
	}
	if args.nameLockAction == output.NameLockDisable {
		result.Impact += " Future generation will allocate names without retained reservations."
	}
	return result, nil
}
