# Internal reference chains to named hierarchy classifiers

A finite typed internal `$ref` chain may end at an existing named hierarchy parent, intermediate or leaf, including supported aliases to that classifier. A component-root alias may bridge into that chain. The generated type remains the canonical physical classifier; generation does not duplicate declarations or expand the referenced JSON tree.

For example, an internal property `Hop.properties.parent` referring to nullable hierarchy `Root` can be referenced by `ParentAlias` using `#/components/schemas/Hop/properties/parent`. `ParentAlias` is a typealias to `Root?`; an explicit false at the alias or use site narrows it to `Root`. A chain through a named hierarchy alias retains that alias's nearest true/false/inherited intent before applying a consumer override. Root/leaf constructor calls and direct companion serializers remain those of the canonical non-null class.

## Identity and graph boundaries

Only actual named Hierarchy declarations authorize these terminals. Original catalog/schema nodes and source Pointers remain intact. Forward catalog aliases and internal root bridges share the catalog dependency traversal; final declarations confirm every used physical classifier. Nominal recursive parent/leaf fields are allowed because a physical classifier is a recursion barrier. Transparent internal-only cycles remain rejected, and every internal and named alias step still consumes the source-depth budget.

[Intrinsic hierarchy nullability](intrinsic-nullable-hierarchies.md), [nullable aliases](nullable-hierarchy-aliases.md), [nullable members](hierarchy-null-members.md), Known/Unknown dispatch and Patch presence keep their existing contracts. False narrowing does not discard cached source height. Borrowed builtin collections cannot acquire nominal leaves through this chain admission. Shape planners do not accept a root bridge as an allOf/union/discriminator source merely because its Kotlin type is nominal.

[Hierarchy-owned direct nested DTO targets](hierarchy-nested-owners.md) have a separate actual-retained-model contract. [Composition-owned DTO targets](composed-nested-owners.md) have their own unique lexical occurrence contract. Ambiguous declarations, inline hierarchy branches and multiple-parent graphs remain separate. This profile does not invent a physical owner for those nodes.

## Verification and limits

The source limit remains 48; an internal chain's incoming edge, every original hop and every intervening named alias count. Typed fields add their field/use-site edges. Nullable adapters add no source edge. Serialized slots and union/enum members remain capped at 240, with bounded graph work.

The [synthetic fixture](../tests/fixtures/http/hierarchy-reference-chains.json), [core regressions](../core/hierarchy_reference_chains_test.go), [source/ownership guards](../core/hierarchy_reference_chains_guards_test.go), [private identity checks](../internal/schema/hierarchy_reference_chains_test.go) and [native preservation check](../internal/cli/hierarchy_reference_chains_process_test.go) cover canonical parent/leaf aliases, mutual nominal references, inherited false/re-null intent, original source identity, final-owner authorization and cached limits. Fresh Kotlin JSON/HTTP/SSE consumers are required for acceptance; this contract does not establish Android/KMP or complete JSON graph support.
