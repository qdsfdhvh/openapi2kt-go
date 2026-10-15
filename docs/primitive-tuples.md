# Primitive positional-array approximation

This is an initial **F08/Q05** subset, not complete JSON Schema tuple or boolean-schema support. OpenAPI 3.1 nonempty `prefixItems` may appear on otherwise-supported array component roots and typed fields/list-map values, inline request/response containers and typed SSE payloads. OpenAPI 3.0 remains unsupported.

## Mapping and diagnostics

Every original prefix member must be an ordinary builtin String/Boolean/Int/Long/Float/Double schema, including supported ordinary numeric formats and string `uuid`. Supported nullable spellings and descriptive metadata remain available. Ref, enum, const, native, value-class, object/container/composition, default, validation-bearing and unknown member keywords are rejected—even when explicit `items` would override them.

- Without `items`, identical scalar kind, format presence/value and effective nullable intent map to `List<T>`.
- Different supported signatures map to `List<JsonElement>`, not a positional union. Absent versus explicitly empty `format` is a distinct signature, even if both map to String.
- Explicit ordinary scalar `items` wins **after every prefix member has been validated**. Unsupported/boolean tails remain rejected; the historical mixed-prefix plus `items: false` fixture still fails at `prefixItems`.
- Empty/malformed prefix arrays and malformed members fail with original source diagnostics. No source nodes are rewritten or manufactured.

Each projected source emits `W_TUPLE_APPROXIMATION` at its original `prefixItems` Pointer. Lists do **not** enforce positions, tuple lengths or positional member types. A typed List also **narrows the schema's unrestricted tail** when no `items` restricts that tail. For example, two string prefix members produce `List<String>`: an empty list and three strings work, but a Boolean in the unrestricted tail cannot decode. Mixed `List<JsonElement>` accepts arbitrary JSON kinds at any position. These intentional approximations are not a strict JSON Schema validator.

`--deny-warnings` rejects this warning before output mutation; `--allow-warning W_TUPLE_APPROXIMATION` explicitly permits it. Default warning policy still generates files.

## Kotlin and wire

Component roots export native collection typealiases with the configured suffix. Use ordinary `listOf(...)` construction and reified serializers/JSON codecs, not alias constructors or `Alias.serializer()`. The wire remains a JSON array. Root and element nullable intent remain independent. Shared nullable fields, required-nullable inline request arguments and optional request Patch follow the existing [presence contract](request-presence.md).

Existing opt-in array count checks apply on typed consumers; standalone aliases/codecs are not validation barriers. `uniqueItems: true` remains restricted to an explicitly declared eligible scalar `items` source; a prefix-only uniform projection does not implicitly gain uniqueness eligibility. Existing source depth48, checked edges16, constructor/method slots240 and graph budgets are unchanged. Original members are visited linearly before any typed/raw projection; they cannot hide errors or reset source depth.

This increment does not admit boolean value schemas, general structured/ref/composed tuples, schema-internal reference targets or other previously unsupported common-union/hierarchy projections. Ordinary non-prefix arrays and boolean `additionalProperties` are unchanged. Previously rejected admitted tuples now generate with an explicit approximation warning; this does not guarantee binary ABI or broader platform coverage.

## Reproducible checks

- [Synthetic envelope](../tests/fixtures/http/primitive-tuples.json): component aliases, mixed/items priority, nullable/list-map, inline Patch/simple HTTP and SSE.
- [Core contracts](../core/primitive_tuple_test.go), [source/limit guards](../core/primitive_tuple_guards_test.go), [native lifecycle and output preservation](../internal/cli/primitive_tuple_process_test.go).
- [Fresh Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PrimitiveTupleContracts.kt): aliases, constructors, four Json profiles, native JSON values, required/null states, explicit approximation behavior, Patch, controlled HTTP and fragmented still-open SSE cancellation/reuse/coldness.
- `bash scripts/test.sh feature primitive-tuple`: fresh Data/Plain × validation off/on generation/check and Kotlin runtime projects, with cached offline dependencies. Data projects also test `copy`; Plain projects do not acquire it.
- `bash scripts/test.sh http`: includes this consumer alongside existing HTTP contracts.

Independent profiles are not maximum-width/depth Cartesian, minimum-toolchain, Android/KMP or complete first-version evidence.
