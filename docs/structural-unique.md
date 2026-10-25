# Structural uniqueness for typed collection trees

With `--validation`, an array with `uniqueItems: true` can contain finite typed List/Map trees with the already accepted scalar or enum codecs. For example:

```json
{"type":"array","uniqueItems":true,"items":{
  "type":"array","items":{"type":"number","format":"float"}
}}
```

The generated type remains `List<List<Float>>`. Two rows containing the same mapped values conflict. List order matters; dictionary insertion order does not. Null containers, empty containers, missing dictionary keys and null-valued entries remain distinct. At most one identical null-containing tree is allowed. Hash collisions do not make different trees equal.

## Accepted sources and equality

The original source and the actual final linked type must both qualify. Containers are ordinary typed lists or dictionaries, with their existing independent nullability and admitted counts/checks. The accepted leaves are String, Int, Long, Float, Double, Boolean, Uuid, LocalDate, Instant, ordinary open String enums and native numeric/Boolean enums with retained membership checks. Existing finite named/internal scalar and collection references and transparent aliases can participate when the shared typed planner already admits them. Forward aliases are checked after their real backing type is linked; this does not add reference or declaration ownership roles.

Scalar equality follows the existing mapped-value contracts. UUID spelling variants and equivalent Instant offsets/fractions conflict. Floating values compare after Float/Double mapping, including rounding collisions and equal positive/negative zeros; non-finite values fail. Open String enums compare `serialName`, so a known value and `Unrecognized` with the same token conflict. This leaves the vocabulary open at runtime. Native enums retain their existing membership and bound checks. Strings use exact parsed equality without Unicode normalization. These are mapped Kotlin values, not a promise of JSON Schema equality for original string/number tokens.

The emitter builds typed structural keys without reflection, serialization or mutation of caller collections. It normalizes enum payloads and floating zeros through each actual List/Map layer. Private non-inline checks with the same rendered type and exact check body can share one model helper, keeping repeated traversals outside wide JVM constructors. The earlier direct builtin [Map profile](map-unique.md) keeps its existing helper and behavior.

## Defaults, presence and consumers

Already admitted collection field defaults and component/alias factories use the same recursive mapped equality. Each original member is validated first. A duplicate annotation fails at the second outer member's original Pointer, including canonical UUID, timestamp, floating and enum equality. Separate keys do not rewrite emitted default JSON or inherit target/intermediate defaults. Child annotations, reference/composition default roles and integer-literal grammar keep their existing acceptance rules.

DTO construction/copy/decoding, nested consumers, Patch.Value, typed HTTP root preflight/success postflight and SSE DTO payloads retain enforcement. Nullable containers and Patch Absent/Null skip contents. Standalone alias assignment/collection serializers, later caller mutation and arbitrary requestBuilder overrides retain their existing transparency boundaries. No new public wrapper, serializer or option is introduced.

DTO/union/hierarchy/value-class/byte/raw projections do not acquire this equality role. Raw JsonElement/JsonObject trees retain the separate [JSON equality contract](json-unique.md). An erased scalar/enum projection cannot obtain typed equality by spelling: every final leaf is paired with its original source, and native enum membership metadata must remain. Existing discarded-check rejection and original diagnostics are preserved. Validated `false` is identical to omission; missing validation opt-in and malformed flags still fail.

## Resources and compatibility

Source depth/work/cache accounting and the 240-slot JVM constructor limit are unchanged. New typed key traversals count toward the existing 16 checked collection-edge limit, including layers below the checked outer array. Default trees retain their existing node/byte budgets. No cap or compiler heap is raised to accommodate the feature. Independent width/depth probes do not establish a combined maximum or a performance guarantee.

This adds accepted D05/M03/M07/M08/M12/V05 inputs. Previously accepted APIs, wire behavior and output/name-lock formats retain their contracts; no persisted-format migration is required. General nominal/projected uniqueness and broader constraint aggregation remain open.

The [core/default/resource guards](../core/structural_unique_test.go), [source/type/identity guard](../internal/schema/structural_unique_test.go), [native output protection](../internal/cli/structural_unique_process_test.go), extended [fixture](../tests/fixtures/http/map-unique.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MapUniqueContracts.kt) cover this increment. `bash scripts/test.sh feature map-unique` generates Data/Plain × validation projects, runs four Json settings, copy/Patch/default/alias/enum/floating cases and HTTP/live fragmented SSE closure/cold-flow/reuse checks, and separately compiles/constructs 240-field and 16-edge models. JVM checks do not prove Android/KMP or minimum-version support.

[Mapped scalar-field DTO uniqueness](dto-unique.md) extends D05/M03/M07/M12/V05 to actual ordinary named/original-inline DTO constructors with direct builtin scalar getters. Typed API/codec/nullable/default/required/Patch behavior remains; constructor initializer-aware default keys match runtime Float/UUID/Instant equality and retain original value Pointers. Actual source/member/constructor proof and independent shared16384work/8MiB source plus16384 default-key work budgets remain; enforcing getter/type fingerprints use a bounded cached reference while provenance/default keys stay private. General nested/enum/alias/composed/hierarchy/projected/count/native roles and broader platform/graph completion remain separate.
