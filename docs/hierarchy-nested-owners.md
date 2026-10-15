# Hierarchy-owned nested DTO references

A local internal schema Pointer may target an ordinary constructor DTO under a named hierarchy leaf's direct property tree, including array items and typed dictionary values. Finite internal chains and component-root aliases to those targets preserve the existing physical nested classifier. For example, `#/components/schemas/First/properties/node` binds the retained `FirstDto.Node`; a rows item binds the actual `FirstDto.RowsInner`, and an entries value binds `FirstDto.EntriesValue`.

Construct and serialize through that canonical owner or a generated root typealias. No copy of the model or alias-owned child classifier is created. Forward aliases, field/list/map consumers and self/mutual nominal recursion use the same physical declarations. Nearest nullable true/false intent, request Patch and every erased source hop follow the [reference-chain contract](hierarchy-reference-chains.md). Container nullability remains independent from element/value nullability.

## Actual retained ownership

Pointer lookup preserves the original catalog node, source Pointer and direct lexical path roles. A provisional nominal carrier stores those identities without guessing a class name. Shared model construction records each successful Model's original node in an analyzer sidecar. After hierarchy planning, only the retained hierarchy Leaf/Nested declaration tree can authorize a unique constructor-bearing occurrence under the original catalog owner. The final FQName comes from those actual ClassNames and emitted scopes.

Scratch/discarded validation models cannot authorize targets because they are absent from retained declarations. Resolved links are removed before semantic equality, optimization and emission. A target first used after hierarchy planning still resolves from the actual retained occurrence registry; every use independently checks its original lexical path. Final declarations revalidate all used classifiers. Generation does not mutate schema JSON, fabricate catalog entries or build an alternate naming planner.

A union field may consume an eligible nested DTO as a typed property; tag/name probing inspects its original source with the full source budget. Complex common fields remain separately constrained; actual nested union constructor targets have the independent [composition-owned contract](composed-nested-owners.md).

## Boundaries and checks

This profile adds direct ordinary constructor DTOs. [Composition-owned DTO references](composed-nested-owners.md) separately admit unique lexical allOf/union Models, including constructor targets in a hierarchy leaf's supported inline union. Borrowed copies do not replace the original lexical owner. Fixed-field additionalProperties scratch shapes, raw/const-only targets, ambiguous occurrences and general shape expansion remain excluded.

Source depth remains 48, retained ownership work is bounded at 16,384, and constructor/serialized slots remain 240 including generated tag bodies. Nominal recursion is a barrier, while transparent aliases retain complete cached height even after false narrowing. Original target validation is still mandatory; invalid nested formats/constraints fail before output replacement.

The [synthetic fixture](../tests/fixtures/http/hierarchy-nested-owners.json), [core API tests](../core/hierarchy_nested_owners_test.go), [source/slot guards](../core/hierarchy_nested_owners_guards_test.go), [late-use and union-consumer regressions](../core/hierarchy_nested_owner_late_uses_test.go), [private provenance tests](../internal/schema/hierarchy_nested_owners_test.go) and [native preservation check](../internal/cli/hierarchy_nested_owners_process_test.go) cover the contract. Fresh Data/Plain × validation Kotlin consumers must verify constructors/codecs, nominal recursion, nullable/Patch, controlled HTTP and live fragmented SSE closure/reuse before acceptance. Android/KMP, other hosts and full nominal graph ownership remain unverified by this profile.
