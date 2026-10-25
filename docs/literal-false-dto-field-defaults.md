# Consuming field defaults with literal false target members

An ordinary named shared DTO field may own a default object targeting an actual ordinary named shared DTO with [literal false properties](literal-false-fields.md). Direct local references and supported pure target aliases retain the actual target codec. A target may also contain existing [false-reference fields](false-reference-fields.md).

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Target":{"properties":{"blocked":false,"labels":{"type":"array","items":{"type":"string"}}},"default":{"labels":["target"]}},
  "Consumer":{"properties":{"value":{"$ref":"#/components/schemas/Target","default":{"labels":["field"]}}}}
}}}
```

The optional `Consumer.value` initializer decodes its own literal into fresh target state. `Target.default()` independently decodes the target's own annotation; neither annotation is inherited or substituted for the other. A target without a root annotation can still be used by an eligible consuming field. Pure aliases preserve the original source and canonical nominal type.

Optional false members may be absent; every supplied value, including explicit null, fails at the original consuming default member Pointer. Required target keys remain mandatory. A logically nullable target permits a null whole-field annotation after complete schema validation. An optional false member's Kotlin null absence carrier does not confer schema nullability. Required consuming annotations validate fully before initializer suppression, preserving the required constructor argument and descriptor.

With `--validation`, target property counts validate the supplied default object against the actual retained target serializer and original interval. Runtime wire counts and existing `encodeDefaults`/`explicitNulls` policies remain distinct from the annotation's supplied keys. Data structural default comparison and Plain reference comparison retain their existing omission behavior.

The shared default scope binds the actual consumer, field, target and source identities, canonical initializer, complete original literal, required/omission policy and count checks. All actual false getter roles validate their target root factories independently; retaining one role cannot replace another role's missing codec or source proof. The deferred validation barrier rechecks actual literal-false helpers, properties, factories and sources before generation succeeds.

**Migration:** these eligible consuming defaults were previously rejected. Existing within-scope accepted Kotlin APIs, wire behavior and ownership/name-lock formats remain compatible. Consumer root defaults, child/container, inline/internal/composed/hierarchy/directional borrowing and broader alias-own annotations retain their separate eligibility. Original source depth48, cumulative false-source work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 limits remain. Each consumer source is captured once, with each field's complete literal checked separately; entering another default does not reset the shared capture budget.

The [fixture](../tests/fixtures/http/literal-false-dto-field-defaults.json), [core contracts](../core/literal_false_dto_field_defaults_test.go), [actual deferred authority](../internal/schema/literal_false_dto_field_defaults_test.go), [native parity/preservation](../internal/cli/literal_false_dto_field_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseDTOFieldDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-dto-field-defaults` freshly generates Data/Plain with validation off/on, baseline and optimized locked output, four Json configurations, independent root/field factories, required/nullable consumers, controlled HTTP, 240 consuming fields and a 240-slot target, and a separate 32,767-byte field initializer. These probes do not establish general Cartesian or other platform coverage.
