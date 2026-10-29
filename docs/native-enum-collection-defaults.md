# Native enum collection defaults on DTO fields

[直接嵌套 native enum 集合默认值](nested-native-enum-collection-defaults.md)另授予 ordinary direct inline finite List/typed Map 树的 outer-own default，保既有五种 mapped scalar、原 exact default membership、逐层 nullable/counts、required/Patch/discarded 预验证、实际优化/锁身份与原预算。本页单层 grammar 不扩张其他 default 角色。

Direct ordinary DTO arrays and typed dictionaries accept collection defaults whose direct items/values are integer, number or Boolean enums. The Kotlin element types remain Int, Long, Float, Double or Boolean; no enum wrapper or new codec is introduced.

```json
{
  "type": "array",
  "items": {"type": "integer", "enum": [1, 2]},
  "default": [1, 2]
}
```

An optional DTO property receives a typed `List<Int>?` initializer. A dictionary with direct Boolean `additionalProperties` similarly receives a typed `Map<String, Boolean>?` initializer. Missing JSON uses that default; explicit null stays null, and supplied collections replace it. Private file-level helpers decode a fresh collection for each initialization, avoiding constructor fields named `kotlin` or `kotlinx` shadowing packages. Kotlin List/Map interfaces remain read-only, without a deep immutability guarantee.

Required fields retain required constructor arguments. Their annotations are still validated. Optional explicitly nullable inline request fields keep `Patch.Absent`, `Patch.Null` and `Patch.Value`; an annotation never turns a missing request key into a default update. Required nullable inline request fields keep their existing `explicitNulls` behavior. Container nullability and member nullability are independent. A null collection default requires explicit container nullability, and a null member requires nullable items/values.

The profile applies to direct fields in ordinary shared/nested DTOs, supported ordinary object allOf models, inline request/response DTOs and SSE payload DTOs. A List/Map of such DTOs retains its existing nominal validation boundary. Later allOf fields keep the existing override rule; every discarded default is validated, and enforcing checks cannot be silently discarded.

## Vocabulary and diagnostics

Every non-null default member must fit its mapped Kotlin scalar and belong to the original enum vocabulary. This applies with validation off as well as on, including required/Patch annotations that emit no initializer and empty or null collection defaults that still require a valid entire leaf schema.

Numeric default membership uses bounded exact source values before Float/Double rounding. For example, vocabulary `[1]` does not admit default `1.000000001` just because both map to the same Float. The existing native enum literal grammar, duplicate rules and numeric precision/exponent bounds remain. The new collection-default profile limits raw vocabularies to 240 members in both modes. Empty/invalid vocabularies fail at the original `/items/enum` or `/additionalProperties/enum`, with original member indices where applicable.

An invalid default fails at `/default/index` or the escaped `/default/<key>`. Dictionary validation visits keys in source order, preserving failure precedence, then sorts successful entries for deterministic emission and identity. Keys such as `$ref`, `type`, `/` and `~` remain opaque data.

With validation off, supplied values retain native scalar behavior and `W_ENUM_NATIVE_APPROXIMATION` at the original leaf vocabulary. With `--validation`, the existing typed native enum metadata checks non-null supplied members in constructors, data-class copies, decoding, HTTP results and SSE payloads. Checked sources stop emitting that approximation warning. Nullable native vocabularies containing one null follow the existing [validation-only null-member contract](native-enum-null-members.md); nullable leaves without a null vocabulary member remain supported in both modes.

## Bounds, identity and compatibility

Validation-enabled `minItems`/`maxItems` and `minProperties`/`maxProperties` constrain both annotations and runtime collections. Existing numeric enum bounds and integer factors also apply to defaults and supplied values. A default that belongs to a vocabulary but violates its bounds fails at the original default-member Pointer. Empty mapped numeric intersections fail at the original enum; floating `multipleOf` remains excluded. Collection or leaf constraints remain unsupported without validation. Int/Long/Boolean enum lists additionally admit [native uniqueness](native-enum-unique.md), retaining membership and numeric checks; Float/Double enum lists use the separate [finite floating uniqueness contract](floating-unique.md). Scalar spelling alone does not grant enum equality eligibility.

Effective defaults and retained checks participate in deduplication, hoisting, merging and name-lock identity. Dictionary key order does not change a frozen default; changing an effective default value cannot reuse a frozen different shape. Existing helpers and accepted non-enum collection outputs remain unchanged, and no persisted format migration is needed.

These collection fields previously failed with `E_SCHEMA_UNSUPPORTED` at the leaf enum. This is an additive M01/M03/M04/M07/M08 capability with existing V05/M12 and A03/A05/A08 consumers, not complete matrix coverage.

## Boundaries and verification

Plain String/typeless enum leaves have a separate [collection-default contract](string-enum-collection-defaults.md) retaining their lexical open enums. Null-only enum leaves, references, composition wrappers, nested collection leaves outside the linked direct-inline native profile, const/native/value-class mappings and pinned union/hierarchy fields retain their separate exclusions. Root and reference-wrapper defaults use their own contracts; eligible inline or referenced enum element/value annotations follow the [child-default contract](collection-reference-child-defaults.md). General allOf constraint aggregation, broader enum uniqueness, platform coverage and release acceptance remain separate work.

The [fixture](../tests/fixtures/http/native-enum-collection-defaults.json), [core regressions](../core/native_enum_collection_defaults_test.go), [native preservation regression](../internal/cli/native_enum_collection_defaults_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumCollectionDefaultsContracts.kt) define the profile. `bash scripts/test.sh feature native-enum-collection-defaults` generates fresh Data/Plain × validation off/on projects and runs all four Json configurations, required/Patch, bounds/membership/defaults, HTTP and fragmented SSE failure closure/reuse. The full `http` route includes the baseline consumer. These JVM checks do not establish Android/KMP, minimum dependency combinations or other engines/hosts.
