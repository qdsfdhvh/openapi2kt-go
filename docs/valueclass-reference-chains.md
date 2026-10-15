# Internal chains to named primitive value classes

Finite typed schema-internal reference chains and component-root bridges may end at an existing named primitive value class or its supported aliases. The canonical wrapper remains the original component's physical declaration, constructor, `rawValue`, non-null companion serializer and init checks. Aliases export that canonical type; no wrapper, source node, catalog entry or serializer is copied.

For example, `Bridge → Owner/properties/hop → Count` binds the actual named `CountDto`, retaining the internal hop and final named edge. A further internal chain ending at an existing bridge keeps the same canonical wrapper and every erased source hop.

## Identity, nullability and layout

Successful value-class construction records the original node and Pointer. The final shared declaration barrier authorizes the unique retained `PrimitiveValueClass` under that original named owner. Typed carriers retain the canonical component and its `InlineValue` TypeRef, including the underlying primitive. Treating it as a generic Named type would lose JVM layout and HTTP parameter behavior.

An explicit consumer wins over nearest root/internal/named alias true/false intent; absent intent inherits intrinsic wrapper nullability. Canonical constructors and direct serializers remain non-null. Nullable reified codecs handle Kotlin null; false restores the original non-null wrapper and codec. Long/Double wrappers consume two slots when non-null and one boxed slot when nullable. This remains true through root bridges, lists/maps, HTTP method allocation and parameter `rawValue` emission.

Constructor/copy/serde, nullable values and Patch.Value use the original wrapper checks. Shared default-null and inline required-nullable/no-default or optional Patch follow the existing presence contract. Source hops and cached height remain bounded at 48 after false narrowing; a physical wrapper is a nominal barrier, while transparent-only cycles reject. Serialized constructor/HTTP parameter budgets remain 240 slots, independently covering 120 non-null Long versus 240 nullable boxed wrappers. No wider compiler resource guarantee is inferred.

## Boundaries and verification

Only existing supported named primitive wrappers are added. Inline wrapper declarations, disabled wrapper flags, scalar/native/provider/temporal/raw projections, enum/const wrappers and shape borrowing retain separate policies. Invalid source, bad bounds and semantic reference siblings reject at original Pointers before returning artifacts. The original catalog/schema/declaration identities remain unchanged.

The [synthetic fixture](../tests/fixtures/http/valueclass-reference-chains.json), [core binding tests](../core/valueclass_reference_chains_test.go), [source guards](../core/valueclass_reference_chains_depth_test.go), [slot guards](../core/valueclass_reference_chains_guards_test.go), [HTTP layout tests](../core/valueclass_reference_chains_http_test.go), [private identity tests](../internal/schema/valueclass_reference_chains_test.go) and [native output preservation](../internal/cli/valueclass_reference_chains_process_test.go) define this profile. `bash scripts/test.sh feature valueclass-reference-chains` creates fresh Data/Plain × validation projects with Plain optimization flags, four Json codecs, canonical constructors/rawValue/nullable/false/bounds, Patch/controlled HTTP/live fragmented SSE closure/reuse and independent 120-unboxed/240-boxed/48-source probes. [The Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ValueClassReferenceChainsContracts.kt) also runs in the full HTTP suite.

Previously rejected finite internal chains to these named wrappers now generate canonical bindings. Existing accepted Kotlin output and wire behavior retain their contracts. This is not full F04/F07/M06, general graph, Android/KMP or additional host acceptance.
