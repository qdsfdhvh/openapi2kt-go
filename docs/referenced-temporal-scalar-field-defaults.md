# Referenced temporal scalar field defaults

An ordinary DTO property with a direct local `$ref` may declare its own `default` through an **already accepted** finite named/internal scalar reference path when the original terminal is `type: string` with standard `format: date` or `date-time`. The actual resolver type remains `kotlinx.datetime.LocalDate` or `kotlin.time.Instant`, including existing intrinsic nullable roots, aliases and root bridges. This role does not authorize a new reference, root or alias, including direct internal physical temporal leaves that the resolver rejects.

```json
{
  "Day": {"type": "string", "format": "date"},
  "Holder": {"properties": {
    "day": {"$ref": "#/components/schemas/Day", "default": "2024-02-29"}
  }}
}
```

The optional property retains `LocalDate?` and a typed parse initializer. Private file-level helpers keep initializers safe when fields are named `kotlin` or `kotlinx`. The transparent alias, provider serializer, classifier, descriptor and nullable backing remain those of the existing no-default input. There is no new DTO, wrapper, JSON-tree helper, constructor check or serializer barrier. [Builtin scalar defaults](referenced-scalar-field-defaults.md) and [referenced builtin collection defaults](referenced-collection-field-defaults.md) retain their separate profiles.

Object-default properties that follow these existing temporal reference paths use the separate [temporal alias member contract](nested-object-defaults.md#temporal-scalar-alias-members).

## Annotation, source and presence

Only the consuming field's own annotation is admitted. Target, intermediate chain, component/root and container-child defaults remain rejected, even when the consumer also supplies a default. Nullable precedence never implies default inheritance. The nearest explicit true/false intent follows the existing resolver; incidental optional-field nullability does not legalize a null annotation.

Every original source, declaration, reference/cache height and check validates before default planning and optimization. Original physical nodes, real catalog membership, canonical RFC6901 Pointers, actual TypeRef and serializer remain authoritative. Forward refs and root bridges use the resolver's existing linked authority. Null, empty and wrong-kind values cannot mask malformed sources. Errors in a value identify the consuming escaped `/default` Pointer; source failures retain their original location. Warnings and warning policy are unchanged.

Optional ordinary fields initialize and decode a missing key from their default. Supplied values replace it; explicit JSON null remains null. Required fields, including required nullable fields, validate the annotation and keep required constructor arguments without initializers. Optional effectively nullable ordinary inline request fields retain `Patch.Absent`; Null and Value remain separate wire states. `explicitNulls=false` retains existing missing-required-null inference and is not strict required-key validation. Discarded ordinary allOf property annotations validate before suppression; this adds no allOf constraint aggregation.

Ordinary shared/nested DTOs and ordinary inline HTTP request/response/SSE payload DTOs share this role. Collection temporal leaves use the independent [referenced collection field own-default role](referenced-temporal-collection-field-defaults.md). Union/hierarchy fields, composition wrappers/folds, enum/const/native `x-native-type` (including magic `date` and coincident provider names), value classes and nominal DTO/union/hierarchy defaults remain excluded. Temporal bounds and temporal `uniqueItems` are not introduced.

## Literal and identity rules

Values use the existing scalar literal/date grammar and the bounded Instant default grammar: real calendar dates with four-digit years, RFC3339 timestamps with uppercase `T`/`Z`, at most nine fractional digits, and offsets through ±18:00. Invalid leap dates, leap seconds, excess precision and out-of-range offsets fail before emission. Original temporal strings remain in effective initializer identity; timezone-equivalent lexemes are not collapsed. Runtime ISO normalization and invalid-value behavior remain with the existing provider codec.

Effective defaults participate in merge/dedup/hoist/name-lock identity alongside actual type/nullability/check/serializer identity. Different lexemes remain distinct; replaying a changed initializer fails the frozen shape contract. Required/Patch annotations without an initializer add no runtime default identity. Historical no-default fingerprints, persisted lock format/profile/allocations/occurrences and output ownership/CAS/no-op behavior remain unchanged.

## Limits and verification surface

The original input8MiB, source48, checked16, slots240 and source-work16,384 caps remain. Every original annotation, including required/Patch/discarded values, retains the referenced-default 16,384 value-node and 32 KiB compact-original/canonical JSON budgets. These independent bounds do not prove width × depth compilation, additional platforms or minimum dependencies.

The [synthetic fixture](../tests/fixtures/http/referenced-temporal-scalar-field-defaults.json), [actual baseline and own-default core controls](../core/referenced_temporal_scalar_field_defaults_test.go), [source/presence/resource guards](../core/referenced_temporal_scalar_field_defaults_guards_test.go), [optimization/lock/reference guards](../core/referenced_temporal_scalar_field_defaults_identity_test.go), [private original authority checks](../internal/schema/referenced_temporal_scalar_field_defaults_test.go) and [native output/lock preservation](../internal/cli/referenced_temporal_scalar_field_defaults_process_test.go) define the subset. The two original date/date-time negatives in the builtin scalar default source-role table explicitly graduate with their input unchanged; other exclusions remain.

`bash scripts/test.sh feature referenced-temporal-scalar-field-defaults` generates fresh Data/Plain × validation off/on projects. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedTemporalScalarFieldDefaultsContracts.kt) checks four Json settings, actual aliases/classifiers/descriptors, constructor/copy/serde/Patch, optimized aliases, controlled HTTP and fragmented still-open SSE failure closure/client reuse/coldness. Independent scalar source48/slots240 probes coexist with existing builtin checked16/default32KiB probes; temporal constraints are not inferred from them. The complete `http` route generates/checks the fixture and calls the consumer once. `OPENAPI_KT_PREPARE_ONLY=1` performs native generation/check and prepares sources only; it is not compilation/runtime evidence.

**Migration:** previously rejected own defaults on this bounded referenced temporal scalar field role now generate; missing keys may decode to their specified temporal value. Accepted no-default Kotlin APIs/wire keep their contracts. This closes only the referenced standard temporal scalar **field own-default** subitem of D03 (M02/M07 and existing A03/A05/A08 consumers). General D03, reference/aggregation gaps and release/platform acceptance remain open.

[DTO scalar reference getter uniqueness](dto-scalar-reference-unique.md) separately admits already-supported builtin/temporal scalar reference fields under validation, with original chain/owner/actual declaration/type/initializer authority. Field-own defaults reuse the existing deferred default planner and source-role restrictions; consumers inherit no component defaults.
