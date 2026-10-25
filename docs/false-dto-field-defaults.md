# DTO field defaults with false-reference members

An ordinary named shared DTO may give a direct named DTO-reference field its own object default when the target has retained [false-reference fields](false-reference-fields.md). The base profile has no component root default on consumer or target; [independently annotated targets](false-dto-independent-defaults.md) have a separate contract. Pure named aliases to the target preserve the same field behavior.

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Denied":false,
  "Target":{"required":["value"],"properties":{
    "value":{"$ref":"#/components/schemas/Denied","nullable":true}
  }},
  "Consumer":{"properties":{
    "target":{"$ref":"#/components/schemas/Target","default":{"value":null}}
  }}
}}}
```

`ConsumerDto()` decodes the canonical annotation through the actual `TargetDto` serializer. Each construction and missing-field decode produces fresh target state. The target gains no component factory. A field without an annotation retains its existing absent/default-null behavior; it does not inherit another field's annotation.

Logical nullable false members admit only JSON null. Non-null optional false members may only be absent; explicitly supplying null still rejects. Literal required keys must be supplied even when nullable or carrying a separate field default. A required consuming field validates its annotation but keeps a required constructor argument and descriptor entry. Caller JSON settings retain their existing missing/null policy. With `encodeDefaults=false`, DataClass structural equality can omit an unchanged object default; PlainClass reference equality can retain it. Fresh allocation and these existing equality policies both remain.

The planner binds the original consumer field, exact target model/getters, source nodes, final initializer and complete literal nodes. Same-text replacement declarations or source nodes cannot substitute for those identities. Source capture is shared per actual consumer, with field literals and source work charged to the same bounded budget. Source depth48, false-source work16,384, default value16,384 nodes/32 KiB, checked depth16 and JVM slots240 remain. The extra consuming reference occupies its original depth; this does not add a fresh depth budget at the target.

Default decoding uses the established default JSON configuration. Caller settings may omit null keys during re-encoding; independently enabled consumer counts still apply to those wire keys. [Target count annotations](false-dto-field-counts.md) have a separate validation-on profile with actual retained count authority. Ordinary DTO JSON HTTP consumers use the same serializer.

Inline/internal/directional/composed owners, container or nested DTO defaults, reference-wrapper factories, consumer component defaults and other target-default combinations and broader graphs retain their separate boundaries. Previously rejected eligible annotations now generate field initializers; existing accepted Kotlin/wire and naming-lock/ownership formats require no migration.

The [fixture](../tests/fixtures/http/false-dto-field-defaults.json), [core contracts](../core/false_dto_field_defaults_test.go), [actual-source guards](../internal/schema/false_dto_field_defaults_test.go), [native lifecycle](../internal/cli/false_dto_field_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOFieldDefaultsContracts.kt) define this profile. `bash scripts/test.sh feature false-dto-field-defaults` exercises Data/Plain × validation off/on, optimized naming locks, four JSON settings and 240-field consumers/targets. These JVM cases do not establish combined maximum resources or other platform acceptance.
