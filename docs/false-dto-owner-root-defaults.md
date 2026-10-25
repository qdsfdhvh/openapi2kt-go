# Owner-root defaults borrowing false DTO members

An ordinary named Shared DTO may own a root default whose direct reference member targets an actual ordinary named DTO with existing literal/ref false getters or inline literal-false collection getters:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Target":{"required":["rows"],"properties":{
    "blocked":false,"rows":{"type":"array","items":false}
  },"default":{"rows":[]}},
  "Outer":{"properties":{"target":{"$ref":"#/components/schemas/Target"}},
    "default":{"target":{"rows":[]}}}
}}}
```

`Outer.default()` decodes its own canonical literal through the actual owner and target serializers, producing fresh DTO/collection state. The target's factory validates independently and is never inherited by an omitted owner member. A target without its own annotation gains no factory. Component naming order does not affect admission: target factory proof is checked after all component factories are planned.

Each direct reference member retains its original `$ref`/nullable source, Pointer and actual owner getter/type storage. The target retains its actual source/model/helper/fields, required keys, logical nullability, field-own annotations and counts. A required target key must be present even when that field has its own default. Optional direct false getters keep strict absence and reject supplied null; collections retain ordinary presence and reject every supplied false leaf. A nullable reference member may contain null, and a nullable owner may own a null root annotation after the full source walk.

Permission belongs only to this original component-own root and direct member reference. It does not turn the member into a property-own default or grant the target a root annotation. Parent source and complete canonical literal capture, including annotation-looking JSON data keys, remain bound to the actual edge and declarations. Missing/replaced source, getter/type storage, target, helper, count walk or independent factory proof rejects. The source48, cumulative false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 budgets remain unchanged.

The direct member must have no own annotation. [Pure component alias paths](false-dto-owner-alias-defaults.md) have a separate provenance/nullability contract. Internal/inline/composed/hierarchy/directional sources, child/member container references, recursive default graphs and other roles retain their independent eligibility. Existing [consuming field annotations](literal-false-collection-dto-field-defaults.md) and [target-own factories](literal-false-collection-root-defaults.md) remain separate. Boolean `additionalProperties` keeps its existing keyword projection and warning semantics.

**Migration:** eligible owner annotations were previously rejected and now gain a root factory. Existing accepted Kotlin APIs, wire behavior and ownership/name-lock formats remain compatible. This adds no default inheritance or generalized graph permission.

The [fixture](../tests/fixtures/http/false-dto-owner-root-defaults.json), [core/value/source budgets](../core/false_dto_owner_root_defaults_test.go), [bounded fuzzing](../core/false_dto_owner_root_defaults_fuzz_test.go), [actual edge/factory/literal proof](../internal/schema/false_dto_owner_root_defaults_test.go), [native preservation](../internal/cli/false_dto_owner_root_defaults_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOOwnerRootDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature false-dto-owner-root-defaults` generates Data/Plain with validation off/on, baseline/optimized locked output, four Json settings and controlled HTTP. Independent probes retain 240 direct reference fields and an actual 240-argument required constructor, a separate 15-container target and an exact 32,768-byte owner literal. These checks do not establish broader platform or minimum-dependency coverage.
