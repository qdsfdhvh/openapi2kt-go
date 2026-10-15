package schema

import (
	"slices"

	"openapi-kt/internal/diagnostic"
)

// Use-site provenance is separate from semantic types. References contribute
// one schema hop, collection edges contribute one, and nullability adds none.
// Cached alias heights stop at nominal DTO/enum/valueclass identities.
type collectionAliasUse struct {
	target  *component
	depth   int
	pointer string
}

func (a *analyzer) aliasConsumerDepth() *diagnostic.Diagnostic {
	// Context planners copy analyzer by value; this shared provenance map must
	// survive those copies. Sort the inventory so map order cannot select an
	// unstable failure when multiple original uses exceed the budget.
	uses := make([]collectionAliasUse, 0, len(a.aliasUses))
	for use := range a.aliasUses {
		uses = append(uses, use)
	}
	slices.SortFunc(uses, func(a, b collectionAliasUse) int {
		if a.pointer < b.pointer {
			return -1
		}
		if a.pointer > b.pointer {
			return 1
		}
		return a.depth - b.depth
	})
	for _, use := range uses {
		if use.depth+1+use.target.CollectionHeight > MaxSchemaDepth {
			return diagnostic.At("E_SCHEMA_DEPTH", "Collection alias use exceeds the supported schema depth.", use.pointer)
		}
	}
	return nil
}
