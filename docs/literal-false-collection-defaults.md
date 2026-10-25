# Own defaults on inline collections with literal false items

An actual ordinary named shared DTO field with [inline collections ending in literal false items](literal-false-collection-fields.md) may own an outer default:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Holder":{"properties":{
    "rows":{"type":"array","items":false,"default":[]},
    "mapping":{"additionalProperties":{"type":"array","nullable":true,"items":false},"default":{"description":null,"empty":[]}}
  }}
}}}
```

The annotation validates the full original schema before decoding through the actual owner-nested false codec. Empty/nested containers are inhabited; explicitly nullable containers permit null. A supplied false leaf admits no value, including null. The optional presence carrier cannot make a non-null source container or false leaf logically nullable.

An optional field initializes from its own canonical literal with fresh collection state. A required annotation validates fully and then suppresses its initializer, including the default-null initializer on a required nullable parameter. Constructors and descriptors retain that required argument. These containers keep ordinary presence and default-comparison behavior; they do not acquire a direct false field's strict absence policy. Existing Json `encodeDefaults`/`explicitNulls` rules remain.

This profile adds the outer field annotation. [Owner root annotations](literal-false-collection-root-defaults.md) have a separate factory contract and may coexist with independent field annotations. [Consuming ordinary DTO field annotations](literal-false-collection-dto-field-defaults.md) have their own actual-target contract. Child defaults, other borrowed DTO defaults, collection roots, inline/internal/composed/hierarchy/directional owners, tuple and uniqueness retain separate eligibility. An annotation is never inherited. Boolean additionalProperties retains its existing keyword projection and warning semantics.

The final source proof binds the actual field/model/helper, original collection and false-leaf nodes, value node, canonical encoded literal, initializer/required policy and complete literal graph. Data keys such as `description`, `items` and `default` remain ordinary JSON data; schema metadata filtering cannot hide replacement of their nested value nodes. Missing literal proof, canonical mismatches, source changes and substituted typed storage reject. All fields share the original false-source capture budget without resetting it for another annotation.

**Migration:** eligible own annotations were previously rejected and now generate typed initializers. Other accepted Kotlin API/wire behavior and ownership/name-lock formats remain compatible. Original source48, cumulative false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 limits remain; required annotations cannot bypass validation by suppressing their initializer.

The [fixture](../tests/fixtures/http/literal-false-collection-defaults.json), [core contracts](../core/literal_false_collection_defaults_test.go), [canonical/literal authority](../internal/schema/literal_false_collection_defaults_test.go), [native lifecycle](../internal/cli/literal_false_collection_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseCollectionDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-collection-defaults` freshly generates Data/Plain with validation off/on, baseline/optimized locked output, four Json configurations and controlled HTTP. Independent resource probes retain 240 fields, a 15-container field, an actual 240-argument required constructor and a separate 32,766-byte field initializer. These JVM probes do not establish general Cartesian, minimum-dependency or other platform coverage.
