# Composition-owned DTO references

Local schema-internal Pointers may bind ordinary constructor DTOs retained under the original component's allOf, oneOf or anyOf source paths. A target can be an inline object-union variant, its nested property DTO, a non-tuple array item or a typed dictionary value. Root aliases and finite typed reference chains retain the physical constructor and serializer. For example, `#/components/schemas/Composed/allOf/0/properties/node` binds the actual `ComposedDto.Node`; `#/components/schemas/Choice/oneOf/0/properties/node` binds the actual retained `ChoiceDto.First.Node`.

Use the canonical owner for nested classifier access. A root typealias creates no alias-owned namespace. Ordinary object-union variants are nested under their union, while a primitive union's object wrapper is emitted as a peer: an internal node under that wrapper binds `ObjectMixedChoiceDto.Node`, according to its actual declaration scope.

## Ownership and copies

Each reference retains its original catalog, node, source Pointer and lexical path. Shared model construction records source provenance alongside successful Models. After shared declarations are built, the retained Model/variant trees authorize a unique constructor-bearing occurrence under that original lexical component. The classifier comes from actual emitted ClassNames and scopes. Source nodes are not expanded, copied or replaced to infer names.

An allOf borrower can retain its existing copy of an inline nested model. References to the original source still use the source component's canonical model. For example, both `BorrowLeftDto.Node` and `BorrowRightDto.Node` can exist, while an internal reference to `Source/allOf/0/properties/node` continues to use `SourceDto.Node`. Removing the source declaration does not let a borrowed copy take ownership.

If composition products retain multiple occurrences of one source node under its own lexical component, a reference to that node is ambiguous and returns `E_REF_UNSUPPORTED`. Generation does not select a variant arbitrarily. Discarded overrides, validation scratch models, fixed-field additionalProperties, empty/raw or const-only models do not authorize a target. After shared ownership is resolved, a first HTTP use may bind only a registered actual occurrence; a missing or ambiguous occurrence fails before artifacts are returned.

## Types and wire behavior

Nearest nullable true/false intent and all erased internal/named reference hops follow the [reference-chain contract](hierarchy-reference-chains.md). Canonical classes and their direct serializers remain non-null; nullable typed views use the nullable codec. Self/mutual nominal recursion, list/map consumers, request Patch, HTTP and SSE data use the same physical types. Container nullability remains independent from item/value nullability.

This extends typed reference ownership. Existing [object union/allOf](union-allof.md) merging and dispatch rules remain: later field overrides, finite tag/key dispatch and Unknown behavior are approximations rather than exact JSON Schema intersections or oneOf validation. Borrowing an internal DTO reference as an allOf or union schema branch remains excluded; structured common fields and dictionary intersections need their own contracts.

## Limits and evidence

Source depth remains 48, ownership/composition work is bounded at 16,384, and emitted constructor/serialization slots remain 240. Every erased transparent hop remains in cached source height after nullable false narrowing. Original targets and discarded schemas still undergo supported source validation. Transient ownership carriers are removed before alias validation, semantic optimization, name-lock fingerprints and emission.

The [synthetic fixture](../tests/fixtures/http/composed-nested-owners.json), [core contracts](../core/composed_nested_owners_test.go), [source/owner guards](../core/composed_nested_owners_guards_test.go), [private provenance tests](../internal/schema/composed_nested_owners_test.go), [native preservation tests](../internal/cli/composed_nested_owners_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedNestedOwnersContracts.kt) cover canonical ownership, source identity, ambiguity, discarded nodes, source/slot limits and wire behavior. `bash scripts/test.sh feature composed-nested-owners` generates fresh Data/Plain × validation projects with four Json configurations, canonical constructors/codecs, relevant optimization options, Patch/controlled HTTP/live fragmented SSE closure and reuse, plus independent 240-slot and 48-source probes.

[References to original allOf constructor nodes](allof-nominal-terminals.md) separately admit unique nested/named Model terminals and their effective nullable intent. General ambiguous graph ownership, multiple-parent/inline hierarchies, dictionary composition and unexecuted Android/KMP or host/engine combinations remain separate capabilities.
