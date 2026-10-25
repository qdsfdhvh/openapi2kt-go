# Referenced String enum scalar field defaults

[Enum alias members inside object defaults](nested-object-defaults.md#string-enum-alias-members) separately compose already accepted reference paths with an enclosing object annotation, retaining the actual classifier and strict original vocabulary.

[Null vocabulary members](referenced-string-enum-null-member-defaults.md) separately compose this own-default role with already admitted explicitly nullable String enum vocabularies, preserving actual narrowed views and the distinction between null and `"null"`.

An ordinary DTO field with a direct local `$ref` may declare its own `default` when its already accepted reference path ends at a plain, unformatted explicit or typeless String enum with a String vocabulary; one null member is admitted separately under the explicit-nullable contract linked above. Named components, existing internal chains, root aliases and nullable intent retain the resolver's admission rules. A default does not admit a new reference graph or nominal role.

```json
{
  "Status": {"type": "string", "enum": ["on", "off"]},
  "Holder": {"properties": {"status": {"$ref": "#/components/schemas/Status", "default": "on"}}}
}
```

The property keeps its actual lexical `StatusDto` classifier and serializer. An internal enum reference continues to use its original enclosing classifier, such as `SourceDto.Code`, even through exported aliases. The initializer is a private typed helper bound to the final property type after hoisting/deduplication/merging; it decodes the validated String through that type's serializer. Helpers are deduplicated across a file, including nested DTOs borrowing the same enum. No wrapper, replacement enum or reconstructed source classifier is introduced.

## Vocabulary and presence

Non-null annotations must be JSON Strings belonging to the original vocabulary in both validation modes. Comparison preserves decoded String content, including escapes and Unicode; it does not normalize Unicode. Invalid values identify the escaped consuming `/default` Pointer. Supplied unknown Strings remain the actual classifier's `Unrecognized` value and retain their wire spelling, with or without validation. Wrong JSON kinds still fail decoding.

Null annotations require effective schema nullability rather than optional-field nullability. Optional missing fields use the known default member, supplied values replace it and explicit null stays null. Required fields keep required arguments, and optional nullable inline request fields keep `Patch.Absent`, after complete annotation validation. Discarded ordinary allOf annotations also validate before override. The existing `explicitNulls=false` inference for missing required nullable values is unchanged.

Only the consuming ordinary field's own annotation is admitted. Target/intermediate/root/wrapper/child defaults and default inheritance remain excluded. String format, primitive bounds/pattern, const/native/value-class terminals and union/hierarchy field roles retain their separate boundaries, including when the annotation is null. Numeric/Boolean enum scalars use the separate [native reference contract](referenced-native-enum-scalar-defaults.md); finite String enum trees use the [collection reference contract](referenced-string-enum-collection-defaults.md).

## Identity and verification

Actual source/declaration authority, original vocabulary, nearest nullable intent and full source-hop accounting finish before default planning and optimization. Canonical decoded String defaults participate in merge/dedup/hoist/name-lock identity; equivalent JSON escape spellings share identity and different defaults remain distinct. Required/Patch annotations without initializers add no runtime default identity. Historical no-default artifacts, persisted formats and output ownership/CAS retain their contracts. Existing input8MiB/source48/checked16/work16,384/slots240/enum240/default32KiB bounds remain, including the 240-member `E_ENUM_LIMIT` diagnostic.

The [fixture](../tests/fixtures/http/referenced-string-enum-scalar-defaults.json), [core tests](../core/referenced_string_enum_scalar_defaults_test.go), [source authority](../internal/schema/referenced_string_enum_scalar_defaults_test.go) and [native parity/lock/preservation tests](../internal/cli/referenced_string_enum_scalar_defaults_process_test.go) define this increment. `bash scripts/test.sh feature referenced-string-enum-scalar-defaults` creates fresh Data/Plain × validation profiles covering actual classifiers/aliases, file-wide helper reuse, escaped defaults, four Json settings, required/Patch, open unknown Strings, optimized/merged/deduplicated output (including replacement of a borrowed enum's enclosing DTO), controlled HTTP and finite SSE decoding/reuse, plus independent 240-field and 240-member compilation probes. The fixture/consumer also run in `http`. `OPENAPI_KT_PREPARE_ONLY=1` prepares native output without Kotlin compilation/runtime evidence.

**Migration:** formerly rejected own defaults on this bounded reference-field role now generate; missing optional keys use their specified known member. Existing accepted no-default call sites and wire retain their contracts. This closes a D03 subitem (M01/M02/M07/M08 with existing M12/V05 and A03/A05/A08 consumers); general D03, wider platform acceptance and release gates remain open.

[DTO String enum reference getters](dto-string-reference-unique.md) separately admit already-supported plain named/internal/bridge enum sources under validation. Actual original enum and intermediate alias identity/physical types, strict own defaults and open serialName runtime behavior remain; other reference roles retain their contracts.
