# Root defaults with literal false DTO properties

An OpenAPI 3.1 ordinary named shared DTO with [literal false properties](literal-false-fields.md) may own a root default:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "OptionalOnly":{"properties":{"blocked":false},"default":{}},
  "Mixed":{"properties":{"blocked":false,"labels":{"type":"array","items":{"type":"string"}}},"default":{"labels":["seed"]}}
}}}
```

`OptionalOnly.default()` returns a fresh DTO whose `blocked` carrier is absent. `Mixed.default()` decodes the validated original object through the actual model serializer, including fresh collection state. Constructors retain their existing parameters and initializers; a component annotation is an explicit opt-in factory and supplies no consuming-field initializer.

A literal false field admits no supplied value. Omitting an optional field is valid; explicit null, false and every other value fail at the original default member Pointer. A required false field must still be present in a non-null object default, so no such default can satisfy that field. An independently nullable owner may own a null root default, including with required false properties: the whole owner is null. Even a null/empty annotation validates the complete original source schema.

Actual original component/model/helper/property identities authorize only that owner's root annotation. The factory binds the original default node, canonical encoded value and complete literal-source snapshot, independently of the field's absence carrier. Deleting or substituting factory/source proof cannot grant permission. Existing property-count checks and required keys retain their original semantics. [Consuming field defaults](literal-false-dto-field-defaults.md) separately validate the consumer and target. Child, wrapper/alias-own, inline/internal/composed/hierarchy/directional defaults retain their separate eligibility; annotations are not inherited.

**Migration:** eligible own root annotations were previously rejected and now add `default()`. Existing accepted source output, Kotlin call sites, wire policy, name locks and ownership formats remain compatible. Optional false properties continue to omit absence under all `encodeDefaults`/`explicitNulls` combinations and reject explicit null. Existing optimizer eligibility, collision-safe helper/import allocation, source48, shared false-work16,384, value16,384, default32KiB, checked16, JVM slots240 and enum240 limits remain.

The [fixture](../tests/fixtures/http/literal-false-root-defaults.json), [core contracts](../core/literal_false_root_defaults_test.go), [actual factory/source authority](../internal/schema/literal_false_root_defaults_test.go), [native parity/preservation](../internal/cli/literal_false_root_defaults_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseRootDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature literal-false-root-defaults` freshly generates Data/Plain with validation off/on, baseline/optimized locked output, four Json configurations, nullable-owner and fresh factories, controlled HTTP, 240-field constructor/descriptor probes and a separate 32,767-byte actual factory literal. These independent JVM probes do not establish Cartesian, minimum-dependency or other platform coverage.
