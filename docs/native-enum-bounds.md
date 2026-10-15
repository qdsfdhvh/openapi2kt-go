# Native enum membership with numeric bounds

With `--validation`, admitted plain numeric enum schemas can retain numeric bounds together with their mapped membership checks. A schema such as `{"type":"integer","enum":[1,2,3,4],"minimum":2,"multipleOf":2}` accepts 2 and 4 in generated typed consumers. It rejects 1 because of the minimum, 3 because of the factor, and 6 because it is outside the vocabulary. The original vocabulary is retained; the generator does not rewrite it to a filtered schema.

This extends [direct DTO enum checks](native-enum-validation.md) and [typed enum consumers](native-enum-consumers.md). Public Kotlin types and wire encodings remain the existing scalars/typealiases. The combined checks run in DTO constructors/decoders, typed List/Map leaves, HTTP input preflight and decoded results, and SSE data construction. Independent alias assignment or alias serialization alone has no checking barrier. Existing nullable, required-presence and Patch policies remain.

## Supported intersection

| Mapped scalar | Retained numeric constraints |
|---|---|
| Int / Long | Inclusive and exclusive bounds; positive representable integer `multipleOf` |
| Float / Double | Inclusive and exclusive bounds with existing finite mapped-value checks |
| Boolean | Membership only; numeric and string constraints are rejected |

OpenAPI 3.0 uses Boolean exclusive modifiers on the corresponding inclusive bound; 3.1 uses numeric exclusive bounds. Each source first passes the existing numeric literal, precision, exponent and domain checks. Integer intersection comparisons are exact and bounded. Float/Double vocabulary values and bound comparisons use their mapped scalar values, including rounding ties. Existing source-domain validation still runs before mapped intersection checks.

The generator rejects an enum with no non-null vocabulary member satisfying all retained constraints, at the original `/enum` Pointer. This rejection also applies to nullable schemas whose non-null intersection is empty; this profile does not lower them to null-only types. [Null native enum members](native-enum-null-members.md) separately admit one null in explicit nullable numeric/Boolean vocabularies under validation; null is excluded from the non-null domain calculation. A supported optional direct DTO default must be a member of the original source vocabulary and satisfy all retained mapped constraints. A nullable null default retains the existing null policy when the non-null intersection is admitted. Required fields still require explicit presence; request Patch defaults retain `Patch.Absent`.

Floating enum plus `multipleOf` is rejected at `/multipleOf`. Existing floating remainder validation without enums remains available; this profile does not use a Go remainder calculation to infer a Kotlin floating intersection. Roots, references and collection leaves do not gain default support. String enums, native overrides/value-class annotations, general internal/named enum graphs, general scalar composition/enum aggregation, and const intersections remain separate.

[Internal native enum Pointers](internal-native-enums.md) also retains these checks through direct borrowed leaves and root bridges under validation.

## Identity, source and compatibility

Bounds and vocabulary both participate in typed merge/dedup and name-lock shape identity. Different checks cannot share an old checked shape. Persisted format versions are unchanged: the existing enum fingerprint domain already includes the fixed numeric-bound slots. Source nodes, original vocabulary and diagnostic Pointers are borrowed without an expanded schema copy. The existing 48 schema/source depth, 16 checked collection edges, 16,384 expansion work and 240 member/slot limits remain.

Raw/stringification projections, ignored additionalProperties and discarded allOf fields still reject checks that they cannot retain. A later allOf override cannot erase enum membership or bounds. This profile does not extend general allOf aggregation.

Without validation, the previous grammar, Kotlin output and warnings remain. Newly admitted enum-plus-bound inputs were previously rejected and remain rejected without `--validation`. Valid historical enum-plus-minimum inputs now have explicit positive coverage preserving their original schema bytes. Supported inputs retain scalar Kotlin call sites and wire behavior; invalid values are refused by the opted-in checks.

[Core behavior](../core/native_enum_bounds_test.go), [intersection guards](../core/native_enum_bounds_guards_test.go), [reuse and lock identity](../core/native_enum_bounds_optimization_test.go), [original source](../internal/schema/native_enum_bounds_test.go), [native output preservation](../internal/cli/native_enum_bounds_process_test.go), the [synthetic fixture](../tests/fixtures/http/native-enum-bounds.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumBoundsContracts.kt) define the contract. `bash scripts/test.sh feature native-enum-bounds` generates fresh Data/Plain profiles with four Json settings. It checks constructor/decode/default/Patch behavior and actual HTTP/SSE inputs and results. Actual platform/toolchain acceptance and the broader JSON roadmap remain separate.
