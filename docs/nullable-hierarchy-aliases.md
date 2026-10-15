# Nullable aliases to non-null hierarchy classifiers

This profile lets an annotation-only named root reference declare nullable intent for an existing non-null [typed hierarchy](model-semantics.md#typed-discriminator-hierarchies) parent or leaf. It retains the original sealed class, leaf constructor, enum properties, mapping routes and per-layer Unknown codec. It creates a typealias rather than a second hierarchy.

For example, an alias to `Root` with `nullable: true` emits `typealias Alias = Root?`. A chain inherits the nearest declared nullable value: a subsequent false alias emits `Root`, and a later true alias emits `Root?`. Explicit true/false consumer references override inherited alias intent. Required fields remain required even when their value type is nullable; optional fields keep their existing missing/default rules.

## Kotlin and JSON null

The physical classifier is non-null. Its `.serializer()` and constructors remain the canonical hierarchy API; a nullable reified alias codec or explicit nullable serializer adapter handles Kotlin null. JSON null decoded through that nullable codec becomes Kotlin null. This does not add a generated Null subtype and does not turn Unknown into a null carrier.

Known and Unknown non-null values retain the original hierarchy wire rules, including the differences between inner and outer explicit mappings. For nullable aliases, Patch.Absent omits an optional field, Patch.Null sends JSON null, and Patch.Value carries a non-null canonical hierarchy value. List items may use nullable alias types independently from the container's nullability.

## Source and graph identity

Only actual final named hierarchy declarations authorize a nullable alias. Candidate discovery borrows original named source nodes, and full hierarchy analysis still validates graph ownership, grammar, mapping targets, cycle/depth/work, fields and budgets. No synthetic component or copied model is introduced. Every transparent alias edge remains in source depth accounting, including false overrides and cached chains.

Limits remain source depth 48, graph work 16,384, union/enum members 240 and serialized constructor slots 240. Source fixtures and existing non-null hierarchy aliases retain their classifier and wire identity. A previously rejected nullable alias input now generates a nullable typealias; callers selecting the direct canonical serializer still request a non-null value.

## Separate nullable contexts

[Intrinsic nullable hierarchy roots, intermediates and leaves](intrinsic-nullable-hierarchies.md) have a separate contract preserving non-null physical classifiers. [Nullable hierarchy members](hierarchy-null-members.md) separately admit explicit nullable refs/aliases, null members and nullable aliases used as mapping targets while retaining physical non-null child serializers. Their union-null and mapping semantics need separate contracts. Nullable aliases to ordinary DTOs, primitive values and non-discriminator unions keep their own rules. General internal nominal references, inline members and multiple hierarchy parents are separate scopes.

The [synthetic input](../tests/fixtures/http/hierarchy-nullable-aliases.json), [core binding checks](../core/hierarchy_nullable_alias_test.go), [depth/source guards](../core/hierarchy_nullable_alias_guards_test.go) and [native preservation test](../internal/cli/hierarchy_nullable_alias_process_test.go) cover canonical parent/leaf aliases, nearest intent, cumulative source limits and failure preservation. Nullable Kotlin codec and HTTP/Patch/SSE behavior require fresh compilation/runtime evidence; source support does not establish Android/KMP acceptance.
