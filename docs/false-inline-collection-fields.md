# Inline collection fields with false-reference leaves

An ordinary named shared DTO may declare a finite inline List/typed Map tree whose original terminal reference points to an existing [false component or pure alias](false-component-roots.md). This complements [named collection roots](false-collection-references.md) without creating a synthetic component for the field.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Holder":{"required":["rows"],"properties":{
    "rows":{"type":"array","items":{"$ref":"#/components/schemas/Denied"}}
  }}
}}}
```

The example exposes a required `List<DeniedDto>` constructor parameter. Empty lists and maps are valid. A non-null false leaf rejects every supplied value, including JSON null. A logically nullable leaf admits null, and no non-null instance can be constructed. The original resolver's nearest nullable intent applies independently at each container and leaf.

An optional container follows the existing ordinary container policy: its presence carrier is nullable/default-null. It does not use the strict absence serializer belonging to a [direct false field](false-reference-fields.md). Required non-null containers remain required; shared required nullable fields retain their existing constructor/default policy. Actual original type-use codecs enforce leaves during ordinary DTO JSON and HTTP serialization.

Validation-on container counts retain their original per-layer checks. Enforcing `uniqueItems` for these leaves stays outside this profile. Source depth48, cumulative false-source work16,384, checked depth16, JVM slots240 and enum limits remain. Entering a field does not reset the source or checked-depth budget. The retained actual model/property/type/check/count identities and original field/container/leaf sources are independently checked; copied same-text declarations or discarded private records grant no permission.

The base field profile has no own default. [Own outer field defaults](false-inline-collection-defaults.md) have a separate complete literal/initializer contract; the owning component still has no root default. Child defaults, container references, inline/internal/directional/composed/hierarchy owners, tuple projection, literal-false collection syntax and discarded extra-key trees remain separate. Direct literal-false DTO properties have a [separate contract](literal-false-fields.md). Other components' independently valid factories retain their existing contracts. No direct false HTTP root is added.

Previously rejected eligible inline fields now generate ordinary collection APIs. Existing accepted output, codecs, naming-lock and ownership formats require no migration. Data/Plain and dedup/hoist/merge options retain the same nominal false-leaf codec.

The [fixture](../tests/fixtures/http/false-inline-collection-fields.json), [core/role/resource contracts](../core/false_inline_collection_fields_test.go), [actual field/source guards](../internal/schema/false_inline_collection_fields_test.go), [native lifecycle](../internal/cli/false_inline_collection_fields_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseInlineCollectionFieldsContracts.kt) define the profile. `bash scripts/test.sh feature false-inline-collection-fields` generates Data/Plain × validation off/on, baseline and optimized locked packages, four JSON settings, controlled HTTP and independent 240-field/15-container probes. These bounded JVM cases do not establish general Boolean graphs, combined maximum resources or additional platforms.
