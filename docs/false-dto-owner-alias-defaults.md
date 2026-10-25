# Pure DTO aliases in owner-root defaults

An ordinary named Shared DTO's [root default reference member](false-dto-owner-root-defaults.md) may follow a finite pure component alias path to the actual ordinary DTO with existing false getters or literal-false collections:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Target":{"properties":{"rows":{"type":"array","items":false}}},
  "Alias":{"$ref":"#/components/schemas/Target"},
  "Maybe":{"$ref":"#/components/schemas/Alias","nullable":true},
  "Outer":{"properties":{"target":{"$ref":"#/components/schemas/Maybe"}},"default":{"target":null}}
}}}
```

The member uses the existing canonical DTO storage and serializer. The nearest explicit nullable alias overrides farther aliases and terminal intent; an explicit nullable annotation on the member overrides the alias path. Optional constructor presence does not authorize a logically non-null default value of null. Required target keys and field-own annotations still validate, and target factories remain independent of the owner's annotation and component naming order. Omitted members inherit no target default.

Each alias retains its original component/schema, reference, canonical binding, nullable binding and cached height. The value walk counts the complete alias path before traversing the original target schema. Original alias source snapshots are captured once per actual component and reused across members/owners within the same cumulative false-source budget; each edge still validates its own path and immutable snapshots. Name spelling or speculative backing types cannot establish permission. Changed catalog entries, cached bindings/heights/nullability or equal-text original-node substitutions reject at the late edge barrier.

Alias paths contain only original component references and existing nullable/metadata annotations. Alias-own defaults, internal targets, composed/hierarchy/directional owners, child/container references and other roles retain independent eligibility. Existing source48, false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 caps remain. Boolean `additionalProperties` keeps its existing keyword projection and warning policy.

**Migration:** eligible owner annotations were previously rejected and now gain factories. Existing accepted Kotlin API/wire behavior and ownership/name-lock formats remain compatible; alias consumers keep their canonical DTO type.

The [fixture](../tests/fixtures/http/false-dto-owner-alias-defaults.json), [core/nullability/budget controls](../core/false_dto_owner_alias_defaults_test.go), [bounded fuzzing](../core/false_dto_owner_alias_defaults_fuzz_test.go), [original alias/cache authority](../internal/schema/false_dto_owner_alias_defaults_test.go), [native preservation](../internal/cli/false_dto_owner_alias_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOOwnerAliasDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature false-dto-owner-alias-defaults` generates Data/Plain with validation off/on, baseline/optimized locked output, four Json settings and controlled HTTP. Independent probes retain 240 direct reference fields and a 240-argument required constructor, a separate 15-container target, 46 alias hops and an exact 32,768-byte owner literal. These checks do not establish broader platform or minimum-dependency coverage.
