# Borrowed collections with retained union leaves

Finite local schema-internal List and typed Map paths, and root aliases exporting those containers, can reuse an original retained object or JSON-kind union. Original `oneOf`/`anyOf` sources and accepted aliases keep their canonical classifier, variants and serializer. A reference to a nested union uses its lexical owner's classifier and sibling native wrappers; a borrowed container does not add a variant namespace.

For example, a List whose item chain ends at a named tagged union `Node` uses `List<NodeDto>`. A Map ending at an inline string/boolean/null union `Owner.node` uses `Map<String, OwnerDto.Node?>`. Known tags dispatch to original variant constructors. Unknown object tags retain the existing raw `Unknown` variant; this support does not add exact oneOf match-count validation or primitive membership validation.

## Ownership and nullable behavior

Admission requires the original lexical catalog/schema node/Pointer and exactly one actual retained `ObjectUnion` or `PrimitiveUnion` declaration. Original semantic branch nodes, keyed and ordered members, shape-reference alias chains, targets, dispatch and nullable provenance authorize the type. Same-name replacements, copied original nodes, missing or repeated declarations and changed dispatch or Null carriers cannot authorize it. Forward aliases use the existing catalog binding rather than a second classifier predictor. No expanded schema document is created.

Standalone JSON-kind unions may contain a physical non-null Null singleton. Decoding JSON null as the canonical root returns that singleton. An inline union can instead return a nullable typed view of a physical non-null classifier: JSON null then returns Kotlin null. Nearest explicit alias/ref/consumer `nullable: true` or `false` preserves the existing priority. Applying a nullable alias to a canonical Null singleton can collapse it to Kotlin null on decode; this is not carrier-identity round-trip preservation. Direct canonical serializers preserve their own non-null dispatch rules.

Root container nullability, item/value nullability and request Patch presence remain independent. Optional nullable request properties keep `Patch.Absent`, `Patch.Null` and `Patch.Value`; ListMap/Patch/HTTP/SSE use the original union codec. Nominal recursion stops transparent expansion, while preceding named/internal references and container segments still count toward the original source budget.

## Boundaries and compatibility

A folded inline single-object nullable union that emits a Model, raw/scratch/discarded declarations, ambiguous repeated occurrences, allOf product unions and hierarchies need their own source profiles. Their syntax or generated sealed spelling does not grant this role. Original terminal grammar still validates before artifacts are returned; failed native generation preserves old output.

Source depth 48, ownership/source work 16,384, checked collection depth 16, union branch count 240 and JVM slots 240 remain fixed. A collection occupies one constructor slot regardless of its nominal leaf. These are separate resource limits, not a claim that every combination of maximum inputs works on every platform.

Previously rejected borrowed container inputs ending at actual unions now generate canonical types. Historical input examples are retained and explicitly converted into this profile's positive contracts; unrelated exclusions and existing wire policies remain. Structured Map parameters keep the current HTTP projection behavior and diagnostics rather than acquiring a new recursive encoder.

## Verification

The [fixture](../tests/fixtures/http/borrowed-collection-union-leaves.json), [core API cases](../core/borrowed_collection_union_leaves_test.go), [grammar/null/source/resource guards](../core/borrowed_collection_union_leaves_guards_test.go), [late HTTP/union uses](../core/borrowed_collection_union_leaves_late_use_test.go), [original-declaration tests](../internal/schema/borrowed_collection_union_leaves_test.go) and [native failure-preservation test](../internal/cli/borrowed_collection_union_leaves_process_test.go) define the contract.

`bash scripts/test.sh feature borrowed-collection-union-leaves` generates fresh Data/Plain × validation projects. The [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BorrowedCollectionUnionLeavesContracts.kt) covers canonical known/Unknown and native Null carriers, recursive lists, nullable Map values, Patch, controlled HTTP, fragmented SSE failure/cancellation/closure/cold reuse, four Json configurations and independent resource probes. The consumer also runs in the full HTTP suite. Full JSON graph support, product/hierarchy collection leaves and Android/KMP or additional host acceptance remain separate.
