# Scalar dictionary defaults on ordinary DTO fields

String schemas with `format: uuid` use [Kotlin Uuid](nullable-uuid.md), including admitted scalar/collection defaults. UUID lexical checks, canonical text and the String-to-Uuid migration apply; original source-role and validation restrictions remain.

Finite direct ordinary DTO dictionaries whose values are further inline List/typed Map containers use the separate [nested builtin default profile](nested-builtin-collection-defaults.md). Per-layer nullable/count/leaf checks and canonical keys are retained; [direct nested temporal defaults](nested-temporal-collection-defaults.md) extend the finite tree, while enum/nominal/ref roles remain excluded.

Ordinary DTO fields with a direct object schema and direct scalar `additionalProperties` accept a JSON object `default`. Supported values are String, Boolean, Int, Long, Float and Double, plus direct String `format: date` / `date-time` mapped to LocalDate / Instant, with independently nullable values. The object may omit `type`, or use an explicit object/nullable-object type. Fixed properties are not part of this profile.

An eligible direct builtin `additionalProperties` schema can also own an independent annotation under the [child-default contract](inline-builtin-collection-child-defaults.md). It never supplies a map initializer or entry.

```json
{
  "type": "object",
  "additionalProperties": {"type": "integer"},
  "default": {"count": 1}
}
```

On an optional field this supplies a typed `Map<String, Int>?` constructor initializer. Missing JSON uses that map; explicit null stays null, and a supplied object replaces it. Each default evaluation produces a fresh map, including an empty default. The public Kotlin `Map` interface is read-only; this is not a deep immutability guarantee.

Required fields have no constructor initializer, although the annotation is still validated. Optional explicitly nullable inline request fields retain `Patch.Absent`: a missing key must not become a default update. `Patch.Null` and `Patch.Value(map)` retain their existing wire behavior. A null container default requires explicit schema nullability; a null member requires a nullable value schema. Nullable containers and nullable values are independent.

Defaults apply to ordinary shared/nested DTOs, inline request/response DTOs and SSE payloads, including the existing ordinary object `allOf` profile. Later fields retain the existing override rule; every discarded original default is still validated. This does not add aggregation of constraints across overridden dictionary fields.

## Validation and identity

With `--validation`, the existing `minProperties`/`maxProperties` and direct primitive value checks validate both annotations and runtime values. Constructor, data-class copy, decoding and typed HTTP/SSE consumption retain those checks. A null skips only its own non-null checks. Empty defaults still require valid value-constraint syntax. Constrained defaults remain unsupported without validation.

Member values must fit the mapped Kotlin scalar. Integer values retain exact range/integrality checks; Float defaults use their mapped Float value before bound checks. String length counts Unicode codepoints. Pattern-default matching uses the existing printable-ASCII boundary; floating `multipleOf` remains unsupported. Count failures point to `/default`; member failures point to the original escaped `/default/<key>`, including keys containing `/` or `~`.

Keys are opaque data: `$ref`, `type` and other schema spellings inside a default are never interpreted as schemas. Validation visits members in source order for diagnostic precedence. Successful defaults sort keys for deterministic emission and normalize mapped numeric values. Changing only key order, integer zero spelling, or equivalent Float spellings does not change model/default identity. Changing an effective key/value or retained check changes the shape used by model reuse and hoisted name locks; existing lock-change rejection still applies.

Private file-level JSON helpers avoid constructor package-name shadowing and large vararg initializers. Existing input, source-depth and JVM constructor limits remain unchanged.

## Date and timestamp values

```json
{
  "type": "object",
  "additionalProperties": {"type": "string", "format": "date-time"},
  "default": {"stamp": "2024-02-29T12:34:56.123456789+02:30"}
}
```

An optional ordinary DTO field receives `Map<String, Instant>?` with a typed initializer; `format: date` similarly uses `Map<String, LocalDate>?`. Missing fields use the default map, explicit null remains null, and supplied maps replace it. Required annotations are validated without supplying a constructor initializer. Nullable inline request fields retain `Patch.Absent`, `Patch.Null` and `Patch.Value(map)` even when a default is present. Values and containers have independent nullability, and empty/default-null maps follow the rules above.

Generation validates original dates and timestamps before emitting a default. Timestamp defaults use the same common Go/Kotlin form as [array defaults](primitive-array-defaults.md#date-and-timestamp-elements): four-digit year, uppercase `T`/`Z`, two-digit time fields, at most nine fractional digits after a dot, and `Z` or an offset within ±18:00. Comma fractions, single-digit hours, excess precision and oversized offsets are rejected. An invalid temporal value or JSON kind fails at its escaped `/default/<key>`, including required and Patch annotations; members are checked in source order. Empty maps still validate the whole value schema.

Private file-level JSON helpers decode fresh typed maps through the existing temporal serializers. Constructor fields named `kotlin` or `kotlinx` cannot shadow them. Instant serialization normalizes timestamps to ISO UTC; exact source spelling is not a wire guarantee. Default identity sorts keys and retains source temporal strings, so equivalent timestamp spellings are not promised to share a name-lock fingerprint. No persisted lock-format change is needed.

With `--validation`, `minProperties`/`maxProperties` validate both default annotations and supplied typed maps, including constructor/data-class copy/decoding and typed HTTP/SSE paths. Temporal value String/numeric constraints remain unsupported; a String source type does not imply date ordering or string-length checks. Native-date markers, references, enums and consts retain their separate exclusions; finite direct nested temporal field trees use the linked [contract](nested-temporal-collection-defaults.md).

## Boundaries and compatibility

Root defaults, defaults on `$ref` or composition wrappers, union/hierarchy fields, fixed-property objects, and nested/nominal/const/native dictionary values remain outside this profile. Collection-element/value annotations follow the separate child contract above. Direct numeric/Boolean and plain String enum values have separate [native](native-enum-collection-defaults.md) and [String](string-enum-collection-defaults.md) collection-default contracts. Boolean `additionalProperties` and untyped JSON dictionaries do not gain defaults. General reference graphs and constraint aggregation are separate capabilities.

These field schemas previously failed with `E_SCHEMA_UNSUPPORTED`. Existing accepted inputs retain their generated API, wire behavior and name-lock format. This extends M01/M02/M04/M07 and the existing typed A03/A05/A08 paths; supported opt-in checks use V05/M12 without claiming complete matrix coverage.

The [plain fixture](../tests/fixtures/http/primitive-map-defaults.json), [checked fixture](../tests/fixtures/http/checked-map-defaults.json), core/native regressions and `bash scripts/test.sh feature primitive-map-defaults` cover Data/Plain, validation off/on, all four JSON configurations, required/Patch, defaults, HTTP and fragmented SSE closure/reuse. These fixtures also cover temporal defaults, independent nullable values, empty/null maps, timezone/precision boundaries, required temporal arguments, ISO normalization and invalid temporal payloads. [Temporal core regressions](../core/temporal_map_defaults_test.go) cover the original empty-date-map admission, escaped member diagnostics, excluded roles, allOf defaults, counts and frozen identity. The [native regression](../internal/cli/primitive_map_defaults_process_test.go) preserves previous output after invalid leap-day and excess-precision defaults. The full `http` route includes the checked consumer. Kotlin/JVM evidence does not establish Android/KMP or other host/engine support.
