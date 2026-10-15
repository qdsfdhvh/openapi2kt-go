# Intrinsic nullable hierarchy components

A validated named hierarchy parent, intermediate or leaf may declare intrinsic `nullable: true`. Its physical Kotlin sealed/data class, constructor and direct `.serializer()` remain non-null; typed references inherit nullable intent and may explicitly narrow with `nullable: false`. This profile extends [nullable hierarchy aliases](nullable-hierarchy-aliases.md) and shares their nullable Kotlin codec rules.

For a nullable `Root` hierarchy, a required property referencing Root uses `Root?`, a false reference uses `Root`, and a transparent root alias inherits Root? unless a nearer alias overrides it. Parent and leaf classifier identities and enum tag types remain canonical. Ordinary model nullable defaults and request presence/Patch rules stay unchanged.

## Hierarchy membership versus consumer nullability

Hierarchy inheritance edges select physical non-null classifiers. A direct mapping to an intrinsically nullable child still selects that child's non-null serializer for a recognized object tag; the child can independently be consumed as a nullable value outside that mapping. A nullable child does not make its parent nullable unless the parent itself declares nullable intent.

[Nullable hierarchy members](hierarchy-null-members.md) have a separate contract for explicit nullable aliases as branch/mapping targets and pure null members. A false alias can narrow an intrinsically nullable child and be used where the existing nominal mapping contract accepts it. That member contract preserves non-null physical inheritance and codecs while carrying member-derived parent nullable intent; no hierarchy Null subtype is introduced.

## JSON null and Unknown

Reified nullable Root? or leaf? codecs decode JSON null to Kotlin null. The direct Root.serializer() requests a non-null hierarchy object; Unknown remains a raw-preserving non-null subtype, distinct from null. Const/enum constructor rules, const-first/compound dispatch, mapping aliases and root-versus-inner routes remain unchanged. Invalid recognized payloads fail even when a nullable codec is used. A const-only or empty mapped nullable leaf keeps its non-null singleton serializer: JSON null is rejected by that direct codec or a false-narrowed consumer, and decoded as Kotlin null only by the nullable adapter. The mapped tag still serializes as an object member.

Request Patch.Value carries a non-null canonical value and Patch.Null sends explicit null. Nullable HTTP bodies/responses and SSE event data use the same TypeRef intent; explicit false versions reject JSON null. SSE cancellation/failure must still close the input stream and preserve client reuse.

## Provenance, limits and compatibility

Intrinsic hierarchy eligibility comes from the original named graph and actual final Hierarchy declarations. Generation does not fabricate schema nodes or duplicate Kotlin classes. Alias/use-site true/false priority and every source hop remain within existing 48-depth and 16,384-work budgets; union/enum/serialized slot limits remain 240. Nullable wrapping does not consume a new source hop or remove a cached reference edge.

This is additive support for previously rejected intrinsic-nullable hierarchy input. Existing accepted non-null hierarchy output and ordinary nullable DTO/enum/primitive behavior remain unchanged. Explicit nullable branches and nullable alias mapping targets follow the separate member contract; multiple parents, inline nodes and general internal nominal ownership remain open.

The [original fixture](../tests/fixtures/http/hierarchy-intrinsic-nullable.json), [core type regressions](../core/hierarchy_intrinsic_nullable_test.go), [mapping/source guards](../core/hierarchy_intrinsic_nullable_guards_test.go) and [native preservation check](../internal/cli/hierarchy_intrinsic_nullable_process_test.go) cover root/intermediate/leaf intent and canonical identities. Fresh Kotlin/runtime evidence is required for nullable serializers, HTTP/SSE and boundaries; this profile does not establish Android/KMP acceptance or a complete JSON graph contract.
