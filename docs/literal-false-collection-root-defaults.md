# Owner defaults containing literal false collections

An ordinary named shared DTO may declare its own root default containing already supported [inline List/typed Map fields ending in literal false array items](literal-false-collection-fields.md):

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Holder":{"required":["rows"],"properties":{
    "rows":{"type":"array","items":false},
    "mapping":{"additionalProperties":{"type":"array","nullable":true,"items":false}}
  },"default":{"rows":[],"mapping":{"description":null,"empty":[]}}}
}}}
```

`Holder.default()` decodes the canonical literal through the actual retained owner serializer and nested false codec. Each call creates fresh collection state. Empty/nested-empty and explicitly nullable containers remain inhabited; every supplied false leaf, including null, rejects. Optional constructor presence does not make a non-null source container logically nullable.

The root annotation creates no field initializer and consumers inherit no default. A required field must occur in an object default even when that field has its own annotation. [Outer field-own defaults](literal-false-collection-defaults.md) validate independently before required suppression; omitted optional fields retain their own initializers. Required and nullable constructor/descriptor policy, normal container presence, direct false strict absence, and Json `encodeDefaults`/`explicitNulls` behavior remain. A nullable owner may own `default: null` after the complete source schema validates.

The value walk borrows only the original collection leaf and Pointer belonging to the actual retained owner/field/helper. Original schema, TypeRef/check snapshots, actual factory and canonical encoded value remain bound to the complete source and literal-node graph. Annotation-looking JSON keys such as `description` are literal data; equal-text replacement of nested null nodes cannot evade provenance validation. Source48, cumulative false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 budgets remain unchanged.

**Migration:** eligible owner annotations were previously rejected and now gain a factory. Existing accepted Kotlin APIs, wire behavior and ownership/name-lock formats remain compatible. [Consuming ordinary DTO field defaults](literal-false-collection-dto-field-defaults.md) have their own actual-target contract. [Owner-root defaults borrowing a direct DTO member](false-dto-owner-root-defaults.md) have a separate edge contract. Child, other borrowed DTO, collection-root, internal/composed/hierarchy/directional, tuple and uniqueness roles retain their independent eligibility. Boolean `additionalProperties` retains its keyword projection and warning behavior.

The [fixture](../tests/fixtures/http/literal-false-collection-root-defaults.json), [core/default budgets](../core/literal_false_collection_root_defaults_test.go), [bounded fuzzing](../core/literal_false_collection_root_defaults_fuzz_test.go), [actual factory/source authority](../internal/schema/literal_false_collection_root_defaults_test.go), [native failure preservation](../internal/cli/literal_false_collection_root_defaults_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseCollectionRootDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-collection-root-defaults` generates Data/Plain with validation off/on, baseline/optimized locked output, four Json settings and controlled HTTP. Independent probes retain 240 fields, a 15-container field, an actual 240-argument required constructor and a separate 32,765-byte root literal. These probes do not establish broader platform or minimum-dependency coverage.
