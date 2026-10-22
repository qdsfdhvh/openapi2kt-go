# Arrays of typed dictionaries with uniqueItems

With `--validation`, `uniqueItems: true` supports array elements that are direct inline typed dictionaries with builtin String, Int, Long, Boolean, UUID, LocalDate or Instant values. Both the dictionary and its values may be nullable. Existing collection aliases and nested consumers can retain the checked array; the item dictionary itself and its value schema must be direct, without references. For example:

```json
{"type":"array","uniqueItems":true,"items":{
  "type":"object","nullable":true,
  "additionalProperties":{"type":"string","nullable":true}
}}
```

The generated type is `List<Map<String, String?>?>`. Equality compares the mapped dictionaries: the same keys with equal values conflict regardless of insertion order. Empty maps, maps with a null-valued entry, and maps with that entry absent differ. At most one null map is allowed. Keys and strings use exact parsed equality without Unicode normalization. Integer and Boolean values use native equality. UUID values compare by parsed Uuid identity; compact/hyphenated and case variants conflict. LocalDate uses date equality, and Instant compares the instant independently of offset or fractional spelling. Distinct nanoseconds remain distinct. Hash collisions do not make distinct dictionaries equal. UUID/Instant equality follows mapped values and can therefore differ from JSON Schema equality of the original JSON strings.

A private file helper builds a set of the supplied maps and compares its size with the list size. It uses Kotlin structural Map equality, without serializing values, sorting the caller's maps or mutating the list. The helper is shared by checked consumers in the file, including wide constructors. This equality profile does not use the separate bounded raw-JSON key algorithm or promise a new runtime size/performance bound.

## Checks, defaults and presence

Retained array counts, dictionary counts and scalar length/pattern/numeric constraints still run at their existing boundaries. DTO construction, Data copy, decoding, nested List/Map consumers, transparent array aliases, Patch.Value, typed HTTP root preflight/success postflight and SSE DTO payloads keep the check. Nullable containers and Patch Absent/Null skip contents. Standalone alias creation, later caller mutation and arbitrary requestBuilder overrides keep their existing transparency boundaries.

Admitted direct/nested/referenced collection field defaults and collection component/alias factories compare the same mapped map values. Each original member is checked first, then sorted object keys and mapped scalar equality keys supply an unambiguous equality key. UUID case/compact/hyphen variants and equivalent Instant offset/fraction spellings cannot conceal duplicates. The equality key is separate from the emitted default JSON. Duplicate defaults fail at the second map's original Pointer; bad map counts or leaf values retain their own Pointers. Existing value/source/byte budgets and integer-literal grammar remain: this does not admit signed-zero or fractional/exponent integer defaults. A runtime JSON integer zero and negative zero can decode to the same Int and therefore conflict.

The profile does not add default roles or inherit target/intermediate annotations. Previously accepted default output is unchanged. Valid new defaults use the existing typed codecs and independent factory/presence rules.

## Scope and compatibility

Eligibility requires both the actual Map type and the original direct dictionary/value schemas. The dictionary source permits type/nullability, `additionalProperties`, admitted `minProperties`/`maxProperties` and ordinary metadata; fixed `properties` or `required` keys do not gain this equality role. Item or value references, nested container values, floating/enum/const/native/value-class values and DTO/composed/raw projections remain outside this typed-map profile. Temporal leaf bounds, folded nullable unions and internal borrowed temporal-collection paths retain their existing source restrictions; this does not add reference/default roles. UUID bounds use the existing canonical mapped text policy. Existing scalar and raw-JSON uniqueness contracts keep their own equality rules. Validated `false` remains an output no-op; malformed flags, wrong-kind keywords, absent validation opt-in and erased/discarded enforcing checks retain their rejection rules.

Existing source depth/work/cache, 16 checked collection edges, 240 JVM slots and output preservation apply. No new semantic check flag, public wrapper, serializer or persisted-format version is introduced. This additively admits previously rejected D05/M03/M07/M12/V05 inputs; already accepted APIs and wire behavior retain their contract. General typed/nominal/projected uniqueness remains open.

The [fixture](../tests/fixtures/http/map-unique.json), [core/default/resource guards](../core/map_unique_test.go), [UUID/temporal Map guards](../core/map_special_unique_test.go), [source/identity guard](../internal/schema/map_unique_test.go), [native output preservation](../internal/cli/map_unique_process_test.go) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MapUniqueContracts.kt) cover this profile. `bash scripts/test.sh feature map-unique` generates fresh Data/Plain × validation projects with four Json settings, Map key order/null/missing/hash collisions, Int/Long extrema, Boolean values, UUID spelling variants, equivalent Instant offsets/fractions and distinct nanoseconds, default factories, Patch, HTTP body/preflight/postflight and fragmented still-open SSE closure/cancellation/cold/client reuse. Checked projects separately compile and construct a 240-field model with mixed UUID/LocalDate/Instant Map values. Plain also uses the existing optimization/name-lock options. JVM evidence does not establish Android/KMP or a minimum-toolchain matrix.
