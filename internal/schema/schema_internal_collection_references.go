package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
	"openapi-kt/internal/naming"
)

// admitInternalSchema establishes a finite transparent tree of direct and
// referenced collections/scalars. Each container has exactly one child; active
// node identities reject cycles across both ref hops and container edges.
// Ordinary DTO leaves retain their actual nominal constructor/serializer and
// stop transparent container traversal. Their original aliases keep full hops;
// other source roles do not acquire permission from Kotlin type spelling.
func (a *analyzer) admitBorrowedOrdinaryDTO(target *resolvedSchema, source string) bool {
	if !directInternalModel(target.Schema, target.Pointer) || a.hierarchySources[target.Schema] {
		return false
	}
	bound := false
	if target.Catalog != nil {
		bound = a.bindNamedChainModel(target, source)
	} else {
		bound = a.bindInternalModel(target, source)
	}
	if bound {
		a.recordBorrowedDTOLeaf(target, source)
	}
	return bound
}

func (a *analyzer) admitInternalSchema(v *document.Value, pointer, source string) *diagnostic.Diagnostic {
	return a.admitInternalSchemaAt(v, pointer, source, 1, map[*document.Value]bool{})
}

func (a *analyzer) admitInternalSchemaAt(v *document.Value, pointer, source string, depth int, active map[*document.Value]bool) *diagnostic.Diagnostic {
	if err := depthCheck(depth, source); err != nil {
		return err
	}
	if v.Kind() != document.Object || active[v] {
		return schemaReferenceUnsupported(source)
	}
	active[v] = true
	defer delete(active, v)
	if enumNominalCandidate(v) {
		// Inline vocabulary leaves still belong to their original lexical
		// declaration. Resolve that exact node instead of planning a new enum
		// under the borrowed container's name.
		target, err := a.lookupSchemaReference("#"+pointer, source)
		if err == nil && target.Catalog == nil && target.Schema == v && a.admitBorrowedNominalLeaf(target, source) {
			return nil
		}
		return schemaReferenceUnsupported(source)
	}
	if a.options.Validation && has(v, "enum") {
		kind, _, err := schemaType(v, pointer)
		if err != nil {
			return err
		}
		if kind == "integer" || kind == "number" || kind == "boolean" {
			return a.terminalInternalScalar(v, pointer, source)
		}
	}
	for _, key := range []string{"$ref", "enum", "const", "x-native-type", "x-kotlin-value-class", "allOf", "oneOf", "anyOf"} {
		if has(v, key) {
			return schemaReferenceUnsupported(source)
		}
	}
	kind, _, err := schemaType(v, pointer)
	if err != nil {
		return err
	}
	var child *document.Value
	var location string
	switch kind {
	case "array":
		if err := checkKeys(v, a.collectionKeys([]string{"type", "nullable", "items"}), pointer); err != nil {
			return err
		}
		var ok bool
		child, ok = v.Get("items")
		if !ok {
			return invalid("Arrays require an items schema.", pointer)
		}
		location = naming.Pointer(pointer, "items")
		if err := object(child, location); err != nil {
			return err
		}
	case "object":
		if !dictionarySchema(v) {
			return schemaReferenceUnsupported(source)
		}
		ap := get(v, "additionalProperties")
		// Open/closed JSON object targets are not typed maps in this profile.
		if ap.Kind() != document.Object {
			return schemaReferenceUnsupported(source)
		}
		child, err = a.dictionaryValue(v, pointer)
		if err != nil {
			return err
		}
		location = naming.Pointer(pointer, "additionalProperties")
	default:
		return a.terminalInternalScalar(v, pointer, source)
	}
	if has(child, "$ref") {
		if err := checkKeys(child, []string{"$ref", "nullable"}, location); err != nil {
			return err
		}
		if _, _, err := schemaType(child, location); err != nil {
			return err
		}
		reference, ok := get(child, "$ref").Text()
		if !ok {
			return invalid("$ref must be a string.", location)
		}
		a.recordBorrowedDTOAliasEdges(reference, location)
		if err := a.visitBorrowedEnumAlias(reference, location, depth+1); err != nil {
			return err
		}
		target, err := a.lookupSchemaReference(reference, location)
		if err != nil {
			return err
		}
		if a.admitBorrowedNominalLeaf(target, location) {
			return depthCheck(depth+max(1, target.Steps), location)
		}
		if target.Catalog != nil {
			return schemaReferenceUnsupported(location)
		}
		_, err = a.resolveInternalReferenceChainAt(target, location, depth+1, active, false)
		return err
	}
	return a.admitInternalSchemaAt(child, location, source, depth+1, active)
}

// Newly borrowed or root-array checked leaves receive hints. Call only with a
// finite admitted tree or a validated alias graph. Hints are private analyzer
// provenance, not semantic equality; old unhinted inputs retain their output.
func (a *analyzer) hintInternalCollectionLeaves(ty TypeRef) {
	if ty.Validation != nil {
		a.emissionHints[ty.Validation] = true
	}
	if ty.Element != nil {
		a.hintInternalCollectionLeaves(*ty.Element)
	}
}
