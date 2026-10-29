package schema

import (
	"strings"

	"openapi-kt/internal/diagnostic"
	"openapi-kt/internal/document"
)

// Root binding retains an indexed physical declaration or explicitly admitted
// transparent scalar source. Borrowed containers keep separate policy. Catalog DFS ensures
// cycles and cached source heights retain their existing policies.
func (a *analyzer) resolveInternalRootAlias(c *component, depth int, visit func(*component, int) *diagnostic.Diagnostic) (bool, *diagnostic.Diagnostic) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(c.Reference, prefix) || !strings.Contains(strings.TrimPrefix(c.Reference, prefix), "/") {
		return false, nil
	}
	source := c.Pointer + "/$ref"
	previousVisit := a.borrowedEnumAliasVisit
	a.borrowedEnumAliasVisit = visit
	defer func() { a.borrowedEnumAliasVisit = previousVisit }()
	target, err := a.lookupSchemaReference(c.Reference, source)
	if err == nil && target.Catalog == nil && !a.bindInternalModel(target, source) {
		if has(target.Schema, "$ref") {
			target, err = a.resolveInternalReferenceChainWithCatalog(target, source, depth, map[*document.Value]bool{}, true, visit)
		} else {
			err = a.admitInternalSchema(target.Schema, target.Pointer, source)
		}
	}
	if err != nil {
		return true, err
	}
	if target.Catalog != nil && target.Deferred == nil {
		target.ModelName = a.physicalDTOName(target.Schema)
	}
	if target.ModelName == "" && target.Deferred == nil && target.CanonicalTemporal == nil && target.CanonicalNativeEnum == nil && target.CanonicalBuiltin == nil && target.CanonicalBigint == nil && target.CanonicalBytes == nil && target.CanonicalCallerNative == nil && target.CanonicalJSON == nil && target.CanonicalCover == nil && target.CanonicalCollection == nil && target.BorrowedCollection == nil && target.BorrowedScalar == nil {
		if err := a.bindBorrowedRootCollection(target, source); err != nil {
			return true, err
		}
		if target.BorrowedCollection != nil {
			return true, a.bindInternalRootAlias(c, target, depth, max(1, target.Steps), target.Nullable)
		}
		if err := a.bindBorrowedRootScalar(target, source); err != nil {
			return true, err
		}
		if target.BorrowedScalar == nil {
			return true, schemaReferenceUnsupported(source)
		}
	}
	return true, a.bindInternalRootAlias(c, target, depth, max(1, target.Steps), target.Nullable)
}

func (a *analyzer) bindInternalRootAlias(c *component, target *resolvedSchema, depth, height int, inherited *bool) *diagnostic.Diagnostic {
	if a.checkedEnumReferenceCandidate(target.Schema) && !a.admitBorrowedNominalLeaf(target, c.Pointer+"/$ref") {
		return schemaReferenceUnsupported(c.Pointer + "/$ref")
	}
	graphHeight := 0
	if target.CanonicalCover != nil {
		graphHeight = a.recursiveJSONHeights[target.CanonicalCover.Schema]
	}
	if err := depthCheck(depth+height+graphHeight, c.Pointer+"/$ref"); err != nil {
		return err
	}
	// Copy only the carrier, never the original schema or a nominal catalog node.
	binding := *target
	c.InternalModelTarget = &binding
	c.AliasHeight = height
	c.BindingNullable = nullableOverride(c.Schema)
	if c.BindingNullable == nil {
		c.BindingNullable = inherited
		if c.BindingNullable == nil {
			nullable, err := a.nominalSourceNullable(target)
			if err != nil {
				return err
			}
			c.BindingNullable = &nullable
		}
	}
	return nil
}
