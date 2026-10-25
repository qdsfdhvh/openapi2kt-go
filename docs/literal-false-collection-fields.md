# Inline collections with literal false items

OpenAPI 3.1 ordinary named shared DTO fields may contain finite inline List/typed Map chains ending in an array `items: false` schema:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Holder":{"required":["rows"],"properties":{
    "rows":{"type":"array","items":false},
    "mapping":{"additionalProperties":{"type":"array","items":false}}
  }}
}}}
```

The required `rows` parameter is `List<Holder.FalseValue>`; the optional mapping uses `Map<String, List<Holder.FalseValue>>?`. Empty lists and map values containing empty lists are inhabited. Every supplied false leaf, including JSON null, rejects. Explicit nullable intent on each container remains independent of its non-null false leaf.

One actual owner-nested sealed, privately constructed `FalseValue` codec is shared with the owner's [direct literal false fields](literal-false-fields.md). Naming collisions retain numeric suffix allocation. Optional containers follow ordinary nullable/default-null presence: they may serialize a null field under existing `encodeDefaults`/`explicitNulls` settings. They do not acquire a direct false field's strict absence serializer. That extra serializer exists only if an actual optional direct false field needs it. Required non-null containers retain required constructor arguments; shared required nullable containers retain their existing default-null policy.

A Boolean `additionalProperties` remains a keyword with its existing JsonObject projection and warning policy. In particular, this profile does not reinterpret `additionalProperties: false` as a typed `Map<String, FalseValue>`. Typed Map values may instead contain arrays whose actual `items` schema is false, as in the example.

The type planner retains the actual original field/container/false-leaf nodes and Pointer/height, owning component/model/nested helper, property/type/count signatures and omission policy. Its temporary collection-element scope admits only that original leaf in that actual field; it creates no fake component and grants no global Boolean schema permission. Retaining another false getter role cannot replace missing source or codec proof. Existing optimizer eligibility and name-lock/ownership formats remain.

**Migration:** eligible fields were previously rejected and now generate typed collection APIs. The original `array-items-boolean` corpus input is retained as an explicit positive graduation with a reviewed complete artifact and actual native/Kotlin checks. Other accepted outputs remain compatible. [Own outer field defaults](literal-false-collection-defaults.md) separately bind their original literal and initializer. [Owner-root defaults](literal-false-collection-root-defaults.md) have a separate factory contract. [Consuming DTO field defaults](literal-false-collection-dto-field-defaults.md) likewise use an independent actual-target contract. Child defaults, collection roots, internal or referenced container sources, inline/internal/composed/hierarchy/directional owners, tuple projection and uniqueness retain separate eligibility. Existing validation-on container counts and source48, cumulative false-work16,384, checked16, JVM slots240 and enum240 limits remain.

The [fixture](../tests/fixtures/http/literal-false-collection-fields.json), [core contracts](../core/literal_false_collection_fields_test.go), [actual source/type authority](../internal/schema/literal_false_collection_fields_test.go), [native parity/preservation](../internal/cli/literal_false_collection_fields_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseCollectionFieldsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-collection-fields` freshly generates Data/Plain with validation off/on, baseline/optimized locked output, four Json configurations, ordinary container presence, strict leaf rejection and controlled HTTP. Its resource project retains 240 fields with an independent 15-container field, an actual 240-argument required constructor and the original corpus input. These probes do not establish full Cartesian, minimum-dependency or other platform coverage.
