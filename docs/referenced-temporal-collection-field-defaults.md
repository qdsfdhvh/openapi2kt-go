# Referenced temporal collection field defaults

An ordinary DTO property with a direct local `$ref` may declare its own outer `default` through an **already accepted** finite named/internal List or typed `Map<String, T>` path whose leaves are standard `type: string`, `format: date` / `date-time`. The resolver keeps actual `kotlinx.datetime.LocalDate` / `kotlin.time.Instant` types and existing aliases, serializers, checks and per-layer nullability. Trees containing the existing builtin6 leaves retain the [referenced builtin contract](referenced-collection-field-defaults.md); plain String enum leaves use the separate [referenced String enum contract](referenced-string-enum-collection-defaults.md). A default does not authorize an otherwise rejected reference or pure schema.

```json
{
  "Days": {"type": "array", "items": {"type": "string", "format": "date"}},
  "Holder": {"properties": {"days": {
    "$ref": "#/components/schemas/Days", "default": ["2024-02-29"]
  }}}
}
```

Named Lists/Maps remain transparent public typealiases with native array/object wire values. Internal collection Pointers, finite chains, aliases and root bridges use their existing admission and linked backing. Nested List/List, List/Map, Map/List, Map/Map and longer accepted finite paths use the same field own-default role, including already accepted named/internal temporal leaf references. Direct physical internal temporal leaves that the resolver rejects remain rejected.

## Source and annotation roles

The role originates only at the consuming ordinary direct-reference field's own annotation. The original physical nodes, canonical RFC6901 Pointers, real catalog entries, linked TypeRef addresses/checks and final unique actual source declarations remain authoritative. Collection edges preserve this collection-own-default role; they do not borrow permission from the independent [temporal scalar field role](referenced-temporal-scalar-field-defaults.md). A coincident Instant/LocalDate type name supplies no source permission: enum/const, native `x-native-type` (including magic `date` and provider FQNames), value-class markers, DTO/union/hierarchy leaves and composition folds remain excluded.

Eligible temporal scalar component/alias factories and finite inline collection factories may coexist with the consuming field's default under the [separate scalar-reference contract](scalar-reference-collection-child-defaults.md). Other target, intermediate-hop and root annotations retain their separate restrictions. Nearest nullable precedence never implies default inheritance. Root/reference composition wrappers, union/hierarchy fields, recursive/nonfinite collection trees, open/raw dictionaries and wider default roles remain outside this profile. The [direct nested temporal grammar](nested-temporal-collection-defaults.md) and general constraint intersections are unchanged.

Every original source and annotation validates before optimization and initializer suppression. Required fields retain required constructor arguments with no initializer, including required nullable fields. Optional effectively nullable inline request fields retain `Patch.Absent`; Null and Value remain separate wire states. Optional ordinary fields construct and decode missing keys from typed defaults; supplied values replace the collection, and explicit null remains null. Incidental optional-field nullability does not authorize a null annotation. Each container and leaf uses the actual resolver's nullable view, including nearest explicit false and the historical direct dictionary policy. `explicitNulls=false` retains existing missing-required-null inference and is not strict required-key validation.

Ordinary shared/nested DTOs and ordinary inline request/response/SSE payload DTOs share this role. Ordinary object allOf keeps later-property overrides and validates discarded original annotations. Discarded constraints without an existing enforcement/aggregation contract fail closed at their original Pointer.

## Values, codecs and identity

Values reuse the existing nested default planner, scalar literal validation and bounded Instant grammar: real four-digit-year calendar dates, uppercase RFC3339 T/Z, at most nine fractional digits, and valid offsets through ±18:00. Invalid leap days, leap seconds, excess precision and oversized offsets fail during generation. Original temporal strings remain part of initializer identity; timezone-equivalent lexemes do not collapse. Runtime wire normalization stays with the existing provider codec.

Original JSON kind, exact Int/Long tokens/ranges, finite mapped Float/Double identity, opaque Map keys and source-order error precedence retain their builtin contracts. Errors identify the consuming original `/default/<index>/<escaped-key>` path. Successful Maps sort keys independently at each layer; arrays retain order. Source trees are borrowed without mutation or reference expansion.

Private typed helpers bind to the final actual rendered property type after optimization, using the original reified collection/leaf serializers and actual aliases. Helper identifiers avoid FQName dots and constructor package shadowing. Each evaluation decodes a fresh tree, including empty containers. Read-only Kotlin List/Map interfaces do not promise deep runtime immutability.

Effective defaults join actual type/alias/nullability/check/serializer identity in deduplication, hoisting, merging and frozen name locks. Canonical Map reordering retains identity; changed temporal lexemes or effective values do not. Required/Patch annotations without initializers add no runtime default identity. No-default fingerprints, persisted lock formats, allocations/occurrences, borrowing/composed ownership and output CAS/no-op contracts are unchanged.

## Checks, budgets and verification surface

With validation enabled, every retained container count checks both the annotation and supplied typed values. Existing builtin counts, eligible innermost scalar uniqueItems and leaf bounds/patterns remain. Temporal `uniqueItems:true` uses the [mapped-value uniqueness contract](temporal-unique.md), including default equality across equivalent Instant offsets/fractions. Temporal bounds stay excluded, including empty/null defaults; validated `uniqueItems:false` remains effect-free. No default annotation may silently filter a check. Disabled validation still rejects constrained sources. Constructor/Data copy/automatic serde, Patch.Value and typed HTTP/SSE keep their original checks; independent alias assignment/codecs, later mutation and caller body overrides remain unchecked boundaries.

Existing limits remain input8MiB, source48, checked collection16, slots240 and source-work16,384. Every original annotation, including required/Patch/discarded values, independently retains 16,384 value nodes, 32 KiB compact-original JSON and 32 KiB canonical JSON limits. These independent probes do not establish maximum width × depth, other platforms or minimum dependencies.

The [synthetic mixed fixture](../tests/fixtures/http/referenced-temporal-collection-field-defaults.json), [Core baseline and types](../core/referenced_temporal_collection_field_defaults_test.go), [source/value/resource guards](../core/referenced_temporal_collection_field_defaults_guards_test.go), [optimization/lock identity](../core/referenced_temporal_collection_field_defaults_identity_test.go), [private original authority](../internal/schema/referenced_temporal_collection_field_defaults_test.go), [native preservation and lock parity](../internal/cli/referenced_temporal_collection_field_defaults_process_test.go) and [FIFO stale CAS/read-only checks](../internal/cli/referenced_temporal_collection_field_defaults_unix_process_test.go) define the subset. Exact former temporal collection/source-role negatives graduate without changing their input or options; excluded roles retain their guards.

`bash scripts/test.sh feature referenced-temporal-collection-field-defaults` freshly generates Data/Plain × validation off/on projects. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedTemporalCollectionFieldDefaultsContracts.kt) covers four Json settings, actual aliases/builtin ABI/descriptors, initializers/missing/null/explicit values, independent trees, constructor/copy/serde/Patch, optimization, HTTP wire/callbacks and still-open fragmented SSE failure closure/cold reuse. Independent source48/checked16/slots240/default32KiB probes keep the original limits. The complete `http` route generates/checks the fixture and calls its consumer once. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check and project preparation only; it is not Kotlin compilation/runtime evidence.

**Migration:** previously rejected own defaults on this bounded referenced temporal collection field role can now generate and supply missing-key values. Accepted no-default Kotlin APIs and wire behavior retain their contracts. This closes only this D03 field own-default subitem (M02/M03/M04/M07, existing M12/V05 and A03/A05/A08 consumers); general D03 remains open.
