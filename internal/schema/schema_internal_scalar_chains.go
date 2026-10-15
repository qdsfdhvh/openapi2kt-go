package schema

import (
	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Transparent internal refs may end at builtin scalars or finite typed
// collections, or an existing internal/direct named DTO or named hierarchy at
// typed uses. Borrowed finite
// collection admission keeps nominal terminals disabled. Admission shares one
// active path across hops and container edges;
// no expansion, catalog mutation or cache hides a later consumer's height.
// Cycles retain the historical excluded-target diagnostic, not nominal SCC policy.
func (a *analyzer) resolveInternalReferenceChain(target *resolvedSchema, source string) (*resolvedSchema, *diagnostic.Diagnostic) {
	return a.resolveInternalReferenceChainAt(target, source, 0, map[*document.Value]bool{}, true)
}

func (a *analyzer) resolveInternalReferenceChainAt(target *resolvedSchema, source string, depth int, active map[*document.Value]bool, nominal bool) (*resolvedSchema, *diagnostic.Diagnostic) {
	return a.resolveInternalReferenceChainWithCatalog(target, source, depth, active, nominal, nil)
}

// Root aliases share catalog DFS dependency/cycle accounting with internal-hop
// admission; ordinary typed uses see the already completed catalog instead.
func (a *analyzer) resolveInternalReferenceChainWithCatalog(target *resolvedSchema, source string, depth int, active map[*document.Value]bool, nominal bool, visit func(*component, int) *diagnostic.Diagnostic) (*resolvedSchema, *diagnostic.Diagnostic) {
	var visited []*document.Value
	defer func() {
		for _, node := range visited {
			delete(active, node)
		}
	}()
	steps := 1
	if err := depthCheck(depth+steps, source); err != nil {
		return nil, err
	}
	var override *bool
	for has(target.Schema, "$ref") {
		v, p := target.Schema, target.Pointer
		if active[v] {
			return nil, schemaReferenceUnsupported(source)
		}
		active[v] = true
		visited = append(visited, v)
		if err := checkKeys(v, []string{"$ref", "nullable"}, p); err != nil {
			return nil, err
		}
		if _, _, err := schemaType(v, p); err != nil {
			return nil, err
		}
		if override == nil {
			override = nullableOverride(v)
		}
		ref, ok := get(v, "$ref").Text()
		if !ok {
			return nil, invalid("$ref must be a string.", p+"/$ref")
		}
		if !nominal {
			a.recordBorrowedDTOAliasEdges(ref, p+"/$ref")
			if err := a.visitBorrowedEnumAlias(ref, p+"/$ref", depth+steps+1); err != nil {
				return nil, err
			}
		}
		next, err := a.lookupSchemaReference(ref, p+"/$ref")
		if err != nil {
			return nil, err
		}
		if next.Catalog != nil {
			if visit != nil && next.Catalog.Reference != "" {
				if err := visit(next.Catalog, depth+steps+1); err != nil {
					return nil, err
				}
				next, err = a.lookupSchemaReference(ref, p+"/$ref")
				if err != nil {
					return nil, err
				}
			}
			ordinaryLeaf := !nominal && a.admitBorrowedNominalLeaf(next, source)
			if !ordinaryLeaf && (!nominal || !a.bindNamedBorrowedCollection(next) && !a.bindNamedBorrowedScalar(next) && !a.bindNamedChainRawTrue(next, source) && !a.bindNamedChainRecursiveCover(next, source) && !a.bindNamedChainCollection(next, source) && !a.bindNamedChainCallerNative(next, source) && !a.bindNamedChainBytes(next, source) && !a.bindNamedChainBigint(next, source) && !a.bindNamedChainBuiltin(next, source) && !a.bindNamedChainNativeEnum(next, source) && !a.bindNamedChainTemporal(next, source) && !a.bindNamedChainModel(next, source)) {
				return nil, schemaReferenceUnsupported(source)
			}
			// Include the final named edge and every erased named-alias hop.
			steps += max(1, next.Steps)
			if err := depthCheck(depth+steps, source); err != nil {
				return nil, err
			}
			if override == nil {
				override = next.Nullable
				if override == nil && !unionNominalCandidate(next.Schema) {
					override = nullableOverride(next.Schema)
				}
				if override == nil && (composedModelCandidate(next.Schema) || unionNominalCandidate(next.Schema)) {
					flag, err := a.nominalSourceNullable(next)
					if err != nil {
						return nil, err
					}
					override = &flag
				}
				if override == nil && next.Catalog.NullableHierarchy {
					flag := true
					override = &flag
				}
			}
			next.Steps, next.Nullable, next.NullableBinding = steps, override, true
			return next, nil
		}
		steps++
		if err := depthCheck(depth+steps, source); err != nil {
			return nil, err
		}
		target = next
	}
	if nominal && a.bindInternalModel(target, source) || !nominal && a.admitBorrowedNominalLeaf(target, source) {
		target.Steps, target.Nullable = steps, override
		return target, nil
	}
	if err := a.admitInternalSchemaAt(target.Schema, target.Pointer, source, depth+steps, active); err != nil {
		return nil, err
	}
	target.Steps, target.Nullable = steps, override
	// Retain an already validated root source for typed metadata probes without
	// changing historical direct borrowed leaf parsing or check-object identity.
	target.BorrowedScalar = a.borrowedScalars[target.Schema]
	target.BorrowedCollection = a.borrowedCollections[target.Schema]
	return target, nil
}

func (target *resolvedSchema) effectiveNullable() (bool, *diagnostic.Diagnostic) {
	if target.Nullable != nil {
		return *target.Nullable, nil
	}
	_, nullable, err := schemaType(target.Schema, target.Pointer)
	return nullable, err
}
