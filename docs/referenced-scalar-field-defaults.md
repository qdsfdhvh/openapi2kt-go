# Referenced scalar field defaults

String schemas with `format: uuid` use [Kotlin Uuid](nullable-uuid.md), including admitted scalar/collection defaults. UUID lexical checks, canonical text and the String-to-Uuid migration apply; original source-role and validation restrictions remain.

[Referenced byte field own defaults](referenced-bytes-field-defaults.md) separately admit scalar byte/binary annotations on existing paths, using signed-array JSON and bounded typed helpers with the same required/Patch/source budgets.

[Referenced String enum scalar field defaults](referenced-string-enum-scalar-defaults.md) separately extend the own-default role to plain String/typeless enum references, retaining actual lexical classifiers, strict annotation vocabulary and open runtime Strings.

[Referenced native enum scalar field defaults](referenced-native-enum-scalar-defaults.md) separately extend the ordinary field own-default role to already admitted numeric/Boolean enum references, preserving strict generation-time vocabulary checks and existing validation admission.

[Referenced standard temporal scalar field defaults](referenced-temporal-scalar-field-defaults.md) separately admit ordinary direct `$ref` fields' own annotations through already accepted date/date-time paths with actual LocalDate/Instant types. Native markers, target/chain/root defaults, collections and other roles retain their exclusions.

An ordinary DTO property with a direct local `$ref` may declare its own `default` when the existing resolver already accepts the named/internal finite path and its terminal is an ordinary builtin String, Boolean, Int, Long, Float or Double schema. Existing aliases, root bridges, source constraints and nearest explicit nullable true/false keep their admission rules. String includes `format: uuid`, including [direct internal leaves and root bridges](root-borrowed-builtin.md#uuid-leaves); annotations use the current native Uuid parser and canonical-text checks. For example:

```json
{
  "Count": {"type": "integer", "format": "int64"},
  "Holder": {"properties": {"count": {"$ref": "#/components/schemas/Count", "default": 2}}}
}
```

The optional field generates `count: kotlin.Long? = 2L`. The declared Count remains a transparent typealias; no new wrapper, serializer or JSON parsing helper is introduced. Actual builtin backing, suffixes, erasure, wire representation and retained validation checks remain those of the existing reference. A default never authorizes a new reference graph or scalar root. In particular, unmarked/false-only aliases to non-null builtin roots still require their existing validation opt-in; intrinsically nullable and own-nullable roots keep their separate accepted profiles.

## Annotation and presence

Only the consuming field's own annotation is accepted through this role. Defaults on the target, an intermediate reference, component/root or composition wrapper remain rejected, including when the consumer declares a default. Eligible collection reference children have a [separate independent annotation contract](scalar-reference-collection-child-defaults.md). Nearest nullable intent does not imply default inheritance. Enum/const, native/provider, value-class, DTO, union and hierarchy terminals and nullable composition folds remain outside this builtin scalar default profile. Temporal fields use the separate profile linked above. Referenced collection fields have their [separate contract](referenced-collection-field-defaults.md).

The original annotation validates before initializer suppression. Optional ordinary fields use the literal for construction and decoding a missing JSON key. A supplied value overrides it; explicit JSON null remains null. A null annotation requires effective schema nullability, rather than incidental optional-field nullability. Required fields retain required arguments with no initializer, including required nullable fields. Optional explicitly nullable inline request fields retain `Patch.Absent`; Null and Value remain separate wire states. The existing `explicitNulls=false` missing-required-null inference is unchanged and does not establish strict required-key validation.

This role applies to ordinary shared/nested DTOs and ordinary inline HTTP request/response/SSE payload DTOs. Ordinary allOf later-property override retains validation of discarded original annotations. Discarded constraints without an aggregation/enforcement contract still reject. Union/hierarchy fields gain no default permission.

## Source and literal rules

Final source/declaration authority, full reference/cache height and checked-depth validation complete before default planning and before optimization. The original nodes, catalog, actual scalar TypeRef and original check object retain authority; default errors identify the escaped consuming `/default` Pointer, while malformed constraints identify their original source keyword. Null or empty values cannot hide an invalid schema/reference. Existing warnings and CLI warning policy remain unchanged.

Values reuse [scalar DTO literal rules](model-defaults.md): Kotlin String escaping, Boolean JSON kind, exact integral-token Int/Long ranges and shadow-safe minimum expressions, finite mapped Float/Double literals and typed suffixes. With validation enabled, retained source checks also apply to the annotation: codepoint String length, printable-ASCII substring portable pattern matching, exact integer bounds/factors and mapped floating bounds including exclusive rounding ties. Floating `multipleOf` with a default remains rejected, even for null. Constraints remain rejected with validation disabled.

DTO construction/copy/automatic serde, Patch.Value and typed HTTP/SSE preserve existing checks. Bare typealias assignment and independent builtin serializers remain unchecked. This feature supplies no new validation barrier.

## Identity and limits

The effective literal participates in existing merge/dedup/hoist/name-lock identity alongside actual type/nullability/check identity. Equivalent mapped Float defaults can reuse a shape; different emitted defaults remain distinct. Required/Patch annotations that emit no initializer add no runtime default identity. Historical no-default fingerprints, lock format/profile/allocations/occurrences and output ownership/CAS are unchanged.

Existing input8MiB, source48, checked16, source-work16,384 and serialized-slot240 limits remain. Each annotation retains the referenced-default profile's 16,384 value nodes and 32 KiB compact-original/canonical JSON budgets, including required, Patch and discarded annotations. These independent bounds do not establish combined width/depth, Android/KMP or minimum-toolchain acceptance.

The [synthetic fixture](../tests/fixtures/http/referenced-scalar-field-defaults.json), [core controls](../core/referenced_scalar_field_defaults_test.go), [source/value/presence guards](../core/referenced_scalar_field_defaults_guards_test.go), [identity tests](../core/referenced_scalar_field_defaults_identity_test.go), [private authority checks](../internal/schema/referenced_scalar_field_defaults_test.go) and [native preservation/lock parity](../internal/cli/referenced_scalar_field_defaults_process_test.go) define the subset. `bash scripts/test.sh feature referenced-scalar-field-defaults` generates fresh Data/Plain × validation off/on consumers for four Json settings, actual alias/builtin ABI and descriptors, constructor/copy/serde/Patch, optimization and HTTP/fragmented SSE failure closure/reuse. Independent source48/checked16/slots240/default32KiB compilation probes are included. `OPENAPI_KT_PREPARE_ONLY=1` runs native generation/check and prepares projects without JVM execution; preparation is not compilation/runtime evidence. The full `http` route generates/checks the fixture and calls the consumer.

**Migration:** previously rejected own defaults on this bounded scalar-reference field role now generate; missing keys may now decode to their specified literal. Accepted no-default Kotlin APIs/wire retain their contracts. This closes only the referenced builtin scalar **field own-default** subitem of D03 (M01/M02/M07 with existing M12/V05 and A03/A05/A08 consumers). General D03 remains open.

[DTO scalar reference getter uniqueness](dto-scalar-reference-unique.md) separately admits already-supported builtin/temporal scalar reference fields under validation, with original chain/owner/actual declaration/type/initializer authority. Field-own defaults reuse the existing deferred default planner and source-role restrictions; consumers inherit no component defaults.
