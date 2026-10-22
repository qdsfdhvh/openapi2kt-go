package schema

import (
	"strconv"
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

func hasObjectCounts(v *document.Value) bool {
	return has(v, "minProperties") || has(v, "maxProperties")
}

// Sources and their locations are ephemeral analyzer planning, never part of
// declaration identity. Each bound applies to the same whole nominal JSON object.
func (a *analyzer) ordinaryObjectCounts(v *document.Value, pointer string, shape *shape, ordinary, constructor bool) (*CollectionValidation, *diagnostic.Diagnostic) {
	type source struct {
		value   *document.Value
		pointer string
		checks  *CollectionValidation
	}
	var sources []source
	eligible := ordinary
	for _, original := range shape.CountSources {
		if !a.originalCountSource(original) {
			return nil, unsupported(original.Pointer)
		}
		if err := depthCheck(original.Depth, original.Pointer); err != nil {
			return nil, err
		}
		sources = append(sources, source{value: original.Schema, pointer: original.Pointer})
	}
	if hasObjectCounts(v) && (len(sources) == 0 || sources[len(sources)-1].value != v || sources[len(sources)-1].pointer != pointer) {
		return nil, rejectObjectCounts(v, pointer)
	}
	// Complete every local grammar/interval check before any source folding.
	for i := range sources {
		s := &sources[i]
		if !eligible || !constructor {
			return nil, rejectObjectCounts(s.value, s.pointer)
		}
		checked, err := a.checkedCollection(TypeRef{Kind: JSONObject}, s.value, s.pointer)
		if err != nil {
			return nil, err
		}
		s.checks = checked.Collection
	}
	var result *CollectionValidation
	for _, s := range sources {
		if result == nil {
			result = &CollectionValidation{}
		}
		for _, minimum := range []bool{true, false} {
			key := "maxProperties"
			if minimum {
				key = "minProperties"
				if n := s.checks.Minimum; n != nil && (result.Minimum == nil || *n > *result.Minimum) {
					result.Minimum, result.MinimumKey = n, key
				}
			} else if n := s.checks.Maximum; n != nil && (result.Maximum == nil || *n < *result.Maximum) {
				result.Maximum, result.MaximumKey = n, key
			}
			if result.Minimum != nil && result.Maximum != nil && *result.Minimum > *result.Maximum {
				return nil, invalid("Minimum collection count exceeds maximum count.", naming.Pointer(s.pointer, key))
			}
		}
	}
	return result, nil
}

// Source metadata must point to the actual immutable document node. This only
// looks up an original path; shape traversal owns all reference expansion work.
func (a *analyzer) originalCountSource(source field) bool {
	if !strings.HasPrefix(source.Pointer, "/") || !hasObjectCounts(source.Schema) {
		return false
	}
	value := a.root
	for _, raw := range strings.Split(source.Pointer[1:], "/") {
		key, err := naming.DecodePointerSegment(raw, source.Pointer)
		if err != nil {
			return false
		}
		switch value.Kind() {
		case document.Object:
			value = get(value, key)
		case document.Array:
			index, e := strconv.Atoi(key)
			if e != nil || index < 0 || index >= len(value.Items()) || strconv.Itoa(index) != key {
				return false
			}
			value = value.Items()[index]
		default:
			return false
		}
	}
	return value == source.Schema
}

// Exact catalog identity distinguishes component roots from nullable nested
// models. Lookup uses the original RFC6901 wire name, never a prefix heuristic,
// reference expansion or an extra context field on semantic declarations.
func (a *analyzer) componentRoot(v *document.Value, pointer string) bool {
	const prefix = "/components/schemas/"
	if !strings.HasPrefix(pointer, prefix) {
		return false
	}
	wire, err := naming.DecodePointerSegment(strings.TrimPrefix(pointer, prefix), pointer)
	if err != nil {
		return false
	}
	component := a.components[wire]
	return component != nil && component.Schema == v && component.Pointer == pointer
}

// Direct root syntax grants provisional count admission only. Ordinary model
// planning must still validate the original fields and retain a constructor;
// nullable intent discovered through composition never grants this permission.
func directNullableDTOCountCandidate(v *document.Value, pointer string) bool {
	kind, nullable, err := schemaType(v, pointer)
	return err == nil && kind == "object" && nullable && !has(v, "$ref") && !has(v, "allOf") && nullableDTOCandidate(v)
}

// Composed nullable DTOs keep the same physical constructor and JSON delegate
// as ordinary counted allOf models. Explicit root intent authorizes this role;
// inferred nullable members and transparent reference wrappers remain separate.
func nullableAllOfDTOCountCandidate(v *document.Value, pointer string) bool {
	kind, nullable, err := schemaType(v, pointer)
	return err == nil && (kind == "object" || kind == "") && nullable && has(v, "allOf") && !has(v, "$ref") && nullableDTOCandidate(v)
}

// The nullable component increment owns exactly its local count source. Borrowed
// sources cannot acquire permission from a coincident direct nullable root.
// ordinaryObjectCounts retains immutable-node, depth and full grammar checks.
func directNullableDTOOwnCounts(v *document.Value, pointer string, shape *shape) bool {
	return directNullableDTOCountCandidate(v, pointer) && len(shape.CountSources) == 1 && shape.CountSources[0].Schema == v && shape.CountSources[0].Pointer == pointer
}

// A component whose nullable shape was discovered from an inline member still
// has no nominal nullable-root codec. Preserve the source keyword diagnostic,
// without rejecting nullable nested DTOs or scanning their independently owned
// fields. Root count diagnostics keep precedence over direct inline members.
func rejectNullableComponentCounts(v *document.Value, pointer string, shape *shape) *diagnostic.Diagnostic {
	if err := rejectObjectCounts(v, pointer); err != nil {
		return err
	}
	for _, source := range shape.CountSources {
		if err := rejectObjectCounts(source.Schema, source.Pointer); err != nil {
			return err
		}
	}
	return nil
}

// Filter values use toString, never the ordinary DTO JSON codec. Scan only
// borrowed schema edges for the new nominal checks; prior primitive/container
// constructor checks remain unchanged. Iterative visitation breaks nominal
// recursion without expanding references or imposing a new nominal graph depth.
func (a *analyzer) rejectDTOCountProjection(v *document.Value, pointer string) *diagnostic.Diagnostic {
	return a.rejectDTOCountProjectionRole(v, pointer, false)
}

// Newly admitted direct nullable root counts cannot be lost through ordinary
// parameter stringification. Historical parameter admission stays unchanged;
// deepObject retains its broader existing count-projection guard.
func (a *analyzer) rejectDTOCountProjectionRole(v *document.Value, pointer string, directNullableRootsOnly bool) *diagnostic.Diagnostic {
	if !a.options.Validation {
		return nil
	}
	queue := []unionBranch{{Schema: v, Pointer: pointer}}
	seen := map[*document.Value]bool{}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		v, p := node.Schema, node.Pointer
		if v == nil || v.Kind() != document.Object || seen[v] {
			continue
		}
		seen[v] = true
		if directNullableRootsOnly && a.componentRoot(v, p) && nullableAllOfDTOCountCandidate(v, p) {
			shape, err := a.modelShape(v, p, 0, true)
			if err != nil {
				return err
			}
			if err := rejectNullableComponentCounts(v, p, shape); err != nil {
				return err
			}
		}
		if !dictionarySchema(v) && (!directNullableRootsOnly || a.componentRoot(v, p) && directNullableDTOCountCandidate(v, p)) {
			if err := rejectObjectCounts(v, p); err != nil {
				return err
			}
		}
		if has(v, "$ref") {
			c, err := a.resolveSchema(text(get(v, "$ref")), p+"/$ref")
			if err != nil {
				return err
			}
			queue = append(queue, unionBranch{Schema: c.Schema, Pointer: c.Pointer})
		}
		for _, member := range v.Members() {
			location := naming.Pointer(p, member.Name)
			switch member.Name {
			case "items", "additionalProperties":
				queue = append(queue, unionBranch{Schema: member.Value, Pointer: location})
			case "properties":
				if member.Value.Kind() == document.Object {
					for _, f := range member.Value.Members() {
						queue = append(queue, unionBranch{Schema: f.Value, Pointer: naming.Pointer(location, f.Name)})
					}
				}
			case "allOf", "oneOf", "anyOf":
				if member.Value.Kind() == document.Array {
					for i, branch := range member.Value.Items() {
						queue = append(queue, unionBranch{Schema: branch, Pointer: naming.Pointer(location, strconv.Itoa(i))})
					}
				}
			}
		}
	}
	return nil
}

func rejectObjectCounts(v *document.Value, pointer string) *diagnostic.Diagnostic {
	for _, key := range []string{"minProperties", "maxProperties"} {
		if has(v, key) {
			return unsupported(pointer + "/" + key)
		}
	}
	return nil
}
