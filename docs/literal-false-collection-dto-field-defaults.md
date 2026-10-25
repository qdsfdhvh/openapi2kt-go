# Field defaults targeting DTOs with literal false collections

An ordinary named shared DTO field may own a default on a direct local reference to an actual retained ordinary DTO containing [inline List/typed Map fields ending in literal false array items](literal-false-collection-fields.md):

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Target":{"properties":{"rows":{"type":"array","items":false}},"default":{"rows":[]}},
  "Consumer":{"properties":{"target":{"$ref":"#/components/schemas/Target","default":{"rows":[]}}}}
}}}
```

The consuming field decodes its own canonical literal through the actual target serializer. Each optional initializer gets fresh DTO/collection state. Empty/nested-empty and explicitly nullable containers retain their existing value rules; every supplied false leaf, including null, rejects. Direct false and referenced false getters can coexist and keep their strict optional absence/null policy. Collection getters keep normal presence semantics.

The target's [own factory](literal-false-collection-root-defaults.md), its [outer field annotations](literal-false-collection-defaults.md) and the consuming annotation validate independently. Consumers inherit no target annotation. Required target members remain required in supplied object literals; required consuming annotations validate fully before initializer suppression. Existing nearest nullable, alias and constructor/descriptor policy remains. Validation-on target counts use the original actual target count plan and original consuming literal Pointers.

Permission exists only within the existing deferred consuming field plan and its exact actual target. The value walk borrows the original false leaf/Pointer and retained field/model/helper TypeRefs; it creates no declaration or global Boolean permission. Missing target source/helper/factory proof, changed getter/check identity or altered consumer literal rejects at the deferred validation barrier. Complete literal capture includes annotation-looking JSON data keys and equal-text nested null-node substitutions. The source48, cumulative false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 budgets remain unchanged.

**Migration:** eligible field-own annotations were previously rejected and now gain typed initializers. Existing accepted Kotlin APIs, wire behavior and ownership/name-lock formats remain compatible. [Owner-root direct reference members without their own annotation](false-dto-owner-root-defaults.md) have a separate actual-edge contract. Other consumer-root annotations, child/collection-root/default inheritance, inline/internal/composed/hierarchy/directional and other source roles retain independent eligibility. Boolean `additionalProperties` keeps its keyword projection and warning policy.

The [fixture](../tests/fixtures/http/literal-false-collection-dto-field-defaults.json), [core contracts and budgets](../core/literal_false_collection_dto_field_defaults_test.go), [bounded fuzzing](../core/literal_false_collection_dto_field_defaults_fuzz_test.go), [deferred target/literal proof](../internal/schema/literal_false_collection_dto_field_defaults_test.go), [native preservation](../internal/cli/literal_false_collection_dto_field_defaults_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseCollectionDTOFieldDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-collection-dto-field-defaults` generates Data/Plain with validation off/on, baseline/optimized locked output, four Json settings and controlled HTTP. Independent probes retain 240 target fields, a 15-container target field, an actual 240-argument required constructor and a separate 32,765-byte consuming initializer. Earlier resource probes remain independent; these checks do not establish broader platform or minimum-dependency coverage.
