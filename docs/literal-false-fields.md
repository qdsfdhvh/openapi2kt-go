# Literal false DTO fields

OpenAPI 3.1 ordinary named shared DTOs may declare properties whose schema is the Boolean literal `false`:

```json
{"openapi":"3.1.0","components":{"schemas":{
  "Optional":{"properties":{"blocked":false}},
  "Required":{"required":["blocked"],"properties":{"blocked":false}}
}}}
```

Each actual DTO owns one public sealed, privately constructed nested `FalseValue`, shared by its literal-false fields. No value inhabits this type. The nested name is allocated against the owner and field-derived declarations; collisions use numeric suffixes such as `FalseValue2`. Existing model suffixes and lexical ownership apply.

A required field exposes a required non-null constructor argument without an initializer. An optional field exposes a nullable argument defaulting to null as an absence carrier. Its strict serializer rejects every supplied JSON value, including explicit null, and its `EncodeDefault.NEVER` policy always omits absence regardless of `encodeDefaults` or `explicitNulls`. The nullable carrier does not make schema false nullable. Required decoding fails for missing keys or any supplied value. An independently nullable owner retains its existing root-null policy.

Actual owner/component, nested helper, property, type, source and required/omission identities authorize this profile. Copying a same-text node or declaration cannot grant a different role. [Root defaults](literal-false-root-defaults.md) separately validate their original owner annotation. [Inline collection fields ending in literal false items](literal-false-collection-fields.md) share the actual nested codec while retaining ordinary container presence. Inline/internal/composed/hierarchy/directional owners, tuple projection, discarded extras and direct false HTTP roots retain their separate eligibility. DTOs containing these nested codecs retain existing optimizer eligibility; unrelated ordinary shapes can still deduplicate, hoist or merge and use name locks.

Constructor annotation arguments use deterministic file-local imports when property names shadow their package qualifiers. Generated property names and wire keys remain intact; alias names avoid declarations, properties and constants in the file. This also preserves literal default text containing annotation-like content.

Source depth48, shared false-source work16,384, value work16,384, default32KiB, checked depth16, JVM slots240 and enum240 limits remain. The shared false-source registry now survives each model-building context, including references to lexically later false components. **Migration:** some previously accepted documents that exceeded this cumulative budget now fail `E_MODEL_LIMIT` at the original component Pointer. The limits themselves have not changed. Output remains intact on failure; accepted within-budget API/wire and persisted ownership/name-lock formats remain compatible. Eligible literal-false fields were previously rejected.

The [fixture](../tests/fixtures/http/literal-false-fields.json), [core contracts](../core/literal_false_fields_test.go), [source authority](../internal/schema/literal_false_fields_test.go), [native lifecycle](../internal/cli/literal_false_fields_process_test.go) and [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseFieldsContracts.kt) define this profile. `bash scripts/test.sh feature literal-false-fields` generates Data/Plain models with validation off/on, baseline and optimized locked output, actual 240 optional/required-field constructor and descriptor probes, four Json profiles and controlled HTTP assertions. These JVM probes do not establish wider role, platform, minimum-dependency or Cartesian resource coverage.
