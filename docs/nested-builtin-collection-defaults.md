# Finite nested builtin collection defaults

[Referenced builtin collection field defaults](referenced-collection-field-defaults.md) extend only the outer ordinary direct `$ref` field role through existing accepted List/Map paths. The direct-inline profile below retains its own grammar; target/chain/child defaults remain rejected.

Direct date/date-time leaves have the separate [nested temporal default contract](nested-temporal-collection-defaults.md), sharing the finite direct-inline grammar and per-layer nullable/count checks. Plain unformatted String enum terminals have the separate [nested String enum default contract](nested-string-enum-collection-defaults.md), preserving actual lexical enums and strict annotation membership with open runtime values; nested enum String bounds/patterns remain rejected. [直接嵌套 native enum 默认值](nested-native-enum-collection-defaults.md)另授予既有 Int/Long/Float/Double/Boolean enum terminal 的 outer-own default，保 native wire、strict source membership 与原 opt-in checks。Other leaf/default roles keep the exclusions below.

Direct ordinary DTO fields may carry a `default` on an already supported finite tree of inline List and typed `Map<String, T>` schemas. Leaves are direct builtin String, Boolean, Int, Long, Float or Double schemas, with their existing format mappings. The four two-container forms are accepted, and longer finite paths follow the same rules:

| Field schema | Kotlin initializer type |
|---|---|
| array → array → integer | `List<List<Int>>` |
| array → dictionary → String | `List<Map<String, String>>` |
| dictionary → array → Boolean | `Map<String, List<Boolean>>` |
| dictionary → dictionary → Float | `Map<String, Map<String, Float>>` |

```json
{
  "type": "object",
  "additionalProperties": {
    "type": "array",
    "nullable": true,
    "items": {"type": ["integer", "null"], "format": "int64"}
  },
  "default": {"rows": [1, null], "empty": [], "nil": null}
}
```

An optional ordinary field exposes `Map<String, List<Long?>?>?` with a typed initializer. Missing JSON uses the default; supplied JSON replaces the field, and explicit null remains null. Each default evaluation decodes a fresh tree through builtin collection/scalar serializers. The public List/Map interfaces are read-only; deep runtime immutability is not promised. Empty containers retain their full generic types.

Nullability is independent at every container and scalar layer. A default null requires explicit nullability on that schema node; optional DTO presence alone does not grant a null annotation. Required fields validate the annotation and retain required constructor arguments without initializers, including required nullable fields. `explicitNulls = false` retains the serializer's existing missing-required-null inference. Optional explicitly nullable ordinary inline request fields retain `Patch.Absent`; a missing key never becomes a default update. Patch.Null and Patch.Value retain their bare JSON wire states. See [request presence](request-presence.md).

## Admission and source fidelity

This profile only grants the field's outer `default` role. Every original schema node is checked, including for an empty or null default. Inner container/leaf `default`, `$ref`, composition, prefixItems, enum, const, DTO, union, hierarchy, native/provider/value-class roles remain outside this builtin profile; direct temporal and plain String enum leaves use the linked extensions. Boolean/open dictionaries and untyped JSON values are excluded. A typed dictionary can omit `type`, or use object/nullable-object; fixed properties are excluded. No previously unsupported pure schema combination is enabled by a default.

The planner borrows the original schemas and JSON values. Leaf admission uses the actual mapped Scalar type and original JSON kind. String/Boolean tokens do not become numbers, and integer defaults retain the existing integral-token and exact Int/Long range rules: decimal/exponent spellings and integer `-0` remain invalid. Float/Double defaults must remain finite in the mapped Kotlin type and use the existing IEEE mapping, rather than promising arbitrary source precision. Temporal parsing or enum vocabularies are separate contracts, including the existing direct [scalar array](primitive-array-defaults.md), [scalar dictionary](primitive-map-defaults.md), [native enum](native-enum-collection-defaults.md) and [String enum](string-enum-collection-defaults.md) defaults.

Invalid values fail at the original `/default/<index>/<escaped-key>/...` Pointer. Each dictionary is visited in original source order, so sorting never changes error precedence. Keys, including `$ref`, `type`, `/` and `~`, remain opaque data. Successful trees sort keys independently at every dictionary level; array order remains significant. Mapped Float/Double values and exact integers are normalized for initializer identity. Reordering keys at any depth or using an equivalent mapped float preserves output/lock identity. Changing an effective value, nullable layer or retained check changes the final shape used by merge, deduplication, hoisting and name locks. Required/Patch annotations that supply no initializer do not invent a runtime default identity. Helpers consume the final actual TypeRef; output text is not substituted to fabricate a classifier.

Defaults work in ordinary shared/nested DTOs and ordinary inline request/response/SSE payload DTOs. Supported ordinary object allOf retains the existing later-field override rule and validates discarded original annotations. Collection/constraint aggregation across overridden contributors remains unsupported; discarded checks cannot silently disappear. Union/hierarchy fields and root/reference/named default roles remain separate.

## Validation and resources

With `--validation`, each admitted layer retains its existing min/max item or property counts. Direct builtin leaves retain eligible String length/pattern and numeric bounds/multiple checks; default annotations and supplied values use those same checks. Innermost scalar lists can retain eligible String/Boolean/Int/Long `uniqueItems`, including at most one nullable null. Unique lists of containers and Float/Double uniqueness remain unsupported. Floating `multipleOf` with a nested default is rejected even when the annotation is empty/null. Counts fail at the corresponding original default container; leaf and duplicate failures retain the original member Pointer. Empty containers still validate constraint grammar and minimum counts. String length counts codepoints; generation-time pattern matching keeps the existing printable-ASCII boundary. Numeric bound arithmetic retains its precision/exponent caps and mapped floating exclusive-bound rules.

Typed constructors, data-class copy, automatic decoding, Patch.Value, HTTP pre/postflight and SSE payloads retain these recursive checks. Null bypasses only its own non-null checks. Bare collection assignment and standalone builtin codecs do not enforce schema bounds. Without validation, constrained schemas remain rejected.

The existing 8 MiB document, source-depth 48, checked-collection-depth 16 and serialized constructor-slot 240 limits are unchanged. New nested annotations additionally have a whole-tree budget of 16,384 JSON value nodes, including containers and nulls, and 32 KiB compact JSON bytes per original default. Keys and scalar contents count toward bytes. Successful canonical JSON is independently limited to 32 KiB. The whole original value is preflighted before allocating canonical containers; exceeded budgets return `E_MODEL_LIMIT` at an original Pointer, and no partial artifacts or warnings are returned. Default value traversal also obeys depth48. These bounds apply to required/Patch/discarded annotations as well as emitted initializers. No cap, compiler heap or network acquisition behavior is increased. Independent width/depth/byte probes do not establish a maximum-width × maximum-depth or Android/KMP guarantee.

## Compatibility and verification

These nested field defaults previously failed with `E_SCHEMA_UNSUPPORTED`. Existing single-layer defaults, pure nested types, Kotlin call sites, wire behavior and persisted lock formats retain their contracts. This closes only the finite direct builtin-tree field-default subitem of D03 and extends M03/M04/M07 with the existing M12/V05 and typed A03/A05/A08 consumers. General defaults and other D03 roles remain open.

The exact formerly rejected empty `List<List<String>>` field is retained as a positive core regression. The [synthetic fixture](../tests/fixtures/http/nested-builtin-collection-defaults.json), [core API/source/resource tests](../core/nested_builtin_collection_defaults_test.go), [guards](../core/nested_builtin_collection_defaults_guards_test.go), [canonical/lock tests](../core/nested_builtin_collection_defaults_identity_test.go) and [native failure preservation](../internal/cli/nested_builtin_collection_defaults_process_test.go) cover acceptance, source ordering, noDrop, required/Patch validation, unchanged output and deterministic identities. Native tests also compare real core/CLI sources and locks, no-op/check/validate bytes/mtimes, read-only output and FIFO-barrier stale CAS preservation.

`bash scripts/test.sh feature nested-builtin-collection-defaults` freshly generates Data/Plain × validation off/on consumers. Its [Kotlin assertions](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedBuiltinCollectionDefaultsContracts.kt) cover four Json configurations, nested nulls/empty trees/key order/native extrema, default omission, required/Patch descriptors, constructor/copy/serde checks, actual dedup/merge/combined hoist/name-lock consumption, typed HTTP and still-open fragmented SSE failure closure/cold reuse. Independent 240-field, source48/checked16 and 32 KiB default compilation probes are included. The full `http` route includes the unconstrained nested consumer. Kotlin/JVM routes do not establish other targets, minimum dependency versions or a complete D03 acceptance matrix.
