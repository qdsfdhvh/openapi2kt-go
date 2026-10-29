# Referenced native enum scalar field defaults

[Native enum alias members in object defaults](nested-object-defaults.md#native-enum-alias-members) separately compose existing reference paths with an enclosing object annotation, preserving original vocabulary, native scalar types and validation gates.

An ordinary DTO field with a direct local `$ref` can declare its own `default` when its already accepted target resolves to a native numeric or Boolean enum scalar: Int, Long, Float, Double or Boolean. The resolver keeps authority over named components, aliases, internal references and nullable intent. In particular, internal native enum references and intrinsic nullable native enum schemas retain their existing validation opt-in; a default does not admit a new reference shape.

```json
{
  "Code": {"type": "integer", "format": "int32", "enum": [1, 2]},
  "Holder": {"properties": {"code": {"$ref": "#/components/schemas/Code", "default": 2}}}
}
```

This produces an optional `code: kotlin.Int? = 2`; the component stays a scalar typealias. No JSON helper, wrapper or enum classifier is introduced. It composes the [referenced scalar own-default role](referenced-scalar-field-defaults.md) with the existing [native enum default membership](enum-defaults.md), using the same terminal planner as [referenced native enum collections](referenced-native-enum-collection-defaults.md).

The consuming ordinary field owns its annotation. Independently admitted [enum component/alias factories](referenced-enum-collection-factories.md) retain their own annotations without inheritance. Composition wrappers, collection children and other unadmitted target/intermediate annotations remain excluded. String enums, const, native markers, value classes, union/hierarchy fields and additional resolver profiles receive no permission from this change.

Non-null defaults must belong to the original vocabulary in both validation modes, using exact numeric comparison before Float/Double mapping; equivalent number spellings compare numerically. Integer defaults retain the existing integral-token grammar and range checks. Literals retain typed suffixes and shadow-safe minimum expressions. Null defaults require effective schema nullability, rather than incidental optional-field nullability. Malformed or unsupported sources reject before null or suppressed initializers can skip value checks. With validation, retained numeric bounds also check the annotation; floating `multipleOf` with a default remains rejected.

Optional missing fields use the literal, supplied values replace it and explicit null remains null. Required fields keep required arguments and optional nullable inline request fields keep `Patch.Absent`, after complete annotation validation. Discarded allOf annotations are validated under the existing no-drop rules. The `explicitNulls=false` missing-required-null inference remains unchanged.

Supplied values retain native scalar wire types. Validation-off preserves the approximation warning and accepts nonmembers. Validation-on typed DTO construction/copy/decoding and HTTP/SSE consumers enforce mapped enum membership and retained bounds. Independent scalar typealias codecs keep scalar decoding in both modes. Defaults use original vocabulary membership even when multiple source numbers map to the same Float.

Effective literals participate in existing merge/dedup/hoist/name-lock identity. Suppressed required/Patch annotations add no runtime default identity. Original source/check authority, no-default fingerprints, persisted formats, output preservation and input8MiB/source48/checked16/work16,384/slots240/enum240/default32KiB budgets are unchanged.

The [fixture](../tests/fixtures/http/referenced-native-enum-scalar-defaults.json), [core tests](../core/referenced_native_enum_scalar_defaults_test.go), [source authority tests](../internal/schema/referenced_native_enum_scalar_defaults_test.go) and [native output/lock tests](../internal/cli/referenced_native_enum_scalar_defaults_process_test.go) define the increment. `bash scripts/test.sh feature referenced-native-enum-scalar-defaults` runs fresh Data/Plain × validation off/on projects for scalar ABI/literals, required/Patch, four Json settings, mapped identity/optimization, checked internal aliases, controlled HTTP and finite SSE decoding/reuse, and independent 240-field and 240-member compilation probes. The consumer is also registered in `http`. `OPENAPI_KT_PREPARE_ONLY=1` prepares native output without Kotlin compilation or runtime evidence.

**Migration:** previously rejected own defaults in this bounded reference-field role now generate, and missing optional keys use their specified literal. Historical accepted no-default APIs/wire retain their contracts. This closes a D03 subitem (M01/M02/M07/M08 with existing M12/V05 and A03/A05/A08 consumers); general D03, wider platform acceptance and release gates remain open.

[DTO native enum reference getters](dto-native-reference-unique.md) separately admit already-supported numeric/Boolean enum scalar paths under validation, preserving actual native export/check/vocabulary and original chain/type/initializer authority. Existing nullable/default-role and no-inheritance contracts remain.
