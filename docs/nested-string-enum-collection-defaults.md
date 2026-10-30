# Directly nested String enum collection field defaults

Component root annotations use the separate [finite nested component factory contract](nested-component-defaults.md); these factories do not become field initializers.

[Directly nested native enum defaults](nested-native-enum-collection-defaults.md) separately cover ordinary direct inline outer field defaults with existing Int/Long/Float/Double/Boolean enum terminals, preserving native wire, exact original default membership, and existing checked membership/bounds. They do not use this page's lexical String enum or Unrecognized API, or extend String bounds/uniqueness.

An ordinary DTO field may declare its own outer `default` on an already accepted finite direct inline List/typed `Map<String, T>` tree whose terminal leaves are plain unformatted explicit String or typeless String enums. List/List, List/Map, Map/List, Map/Map, and longer finite mixed paths use the same rule. A default does not admit an otherwise unsupported schema.

```json
{
  "type": "array",
  "items": {
    "additionalProperties": {
      "type": ["string", "null"],
      "enum": [null, "ready", "waiting", "null"]
    }
  },
  "default": [{"first": "ready", "nil": null}, {}]
}
```

As an optional ordinary `labels` field, this schema retains the actual lexical enum `LabelsInnerValue` and has type `List<Map<String, LabelsInnerValue?>>?`. The leaf retains its original entries, `serialName`, `fromSerialName`, serializer, and `Unrecognized` API; it is not projected to `List<Map<String, String>>`. Missing JSON uses a typed initializer, explicit null remains null, and an explicit collection replaces the whole default collection. Each evaluation of a non-null initializer decodes an independent collection tree through a typed JSON helper bound to the final actual enum TypeRef; empty collections retain full generics and null remains nullable. Public List/Map interfaces do not promise deep immutability.

The original six builtin leaves and date/date-time leaves follow the [nested builtin](nested-builtin-collection-defaults.md) and [nested temporal](nested-temporal-collection-defaults.md) contracts respectively, retaining their literal, codec, and check semantics. [Single-layer String enum collection defaults](string-enum-collection-defaults.md) and their [leaf bounds](string-enum-collection-bounds.md) are separate scopes.

## Original vocabulary, nullability, and field presence

Every non-null default leaf must be a JSON String in the original enum vocabulary with either validation setting. Unknown defaults are rejected at the original member Pointer; at runtime, unknown Strings can still be accepted and written back through the original `Unrecognized` or `fromSerialName`. `--validation` does not close the String enum vocabulary or add native enum membership checks.

Nullable intent is independent at each List, Map, and enum leaf layer. Null in a default requires nullable intent at the corresponding schema node; a plain typeless String/null leaf can have intrinsic nullability under [shared vocabulary inference](implicit-string-enum-null.md). The Kotlin nullable view created by an ordinary optional field alone does not authorize an annotation null. Leaves retain the [String null-member grammar](string-enum-null-members.md): at most one null and at least one String, preserving original member indices and counting null toward the 240 raw member limit. JSON null remains distinct from the vocabulary String `"null"`. Container nullability does not propagate to inner leaves, and inner nullability does not create Patch.

Required fields, including required nullable fields, retain required constructor parameters without initializers after full annotation validation. `explicitNulls=false` retains the serializer's existing missing-required-null inference. Optional explicitly nullable fields in ordinary inline requests remain `Patch.Absent` after full validation; a missing key does not become a default update. Patch.Null and Patch.Value retain bare JSON wire behavior. See [request presence](request-presence.md).

Ordinary shared/nested DTOs and ordinary inline request/response/SSE payload DTOs can consume this field profile. Previously supported ordinary object allOf retains later-field override and validates every original discarded annotation before override or initializer suppression; invalid vocabulary, defaults, constraints, or budgets cannot disappear through override. No collection/enum check aggregation across contributions is added. Union/hierarchy fields retain their existing exclusions.

## Full schema validation and source fidelity

Only an ordinary direct inline field's own outer default is admitted. Empty/null annotations also first validate the entire schema tree, all containers, and the complete original vocabulary. Inner container/leaf defaults, root/named/reference/wrapper/target/chain defaults, and default inheritance are outside this profile. Referencing a DTO that has a valid direct field does not admit defaults on the referenced leaf or its collection root.

The tree uses only previously accepted inline List and typed Map grammar. A typed dictionary may omit `type` or declare object/nullable-object, and may not introduce nonempty fixed properties/required. Open/Boolean dictionaries, untyped JSON, prefixItems, composition wrappers, recursive trees, and general intersections remain excluded. The added terminal is only a plain unformatted String enum; numeric/Boolean enum, native/formatted/provider/const/value-class/DTO/union/hierarchy leaves and referenced/named leaf defaults retain their rejection boundaries.

Validation uses the original JSON kind, vocabulary members, and schema nodes. Errors use the original `/default/<index>/<escaped-key>/...` Pointer. Each Map is checked in source order, and its keys are sorted only after success. Dictionary keys such as `$ref`, `type`, `/`, and `~` are always opaque data, not schemas or references. Array order and original enum String tokens are preserved.

## Validation and resources

With `--validation`, each List's min/maxItems and typed Map's min/maxProperties constrain both the original default and supplied typed values. Empty collections must still satisfy minimum counts; null skips only its own non-null checks. Constructors, Data copy, automatic serialization, Patch.Value, typed HTTP pre/postflight, and SSE payloads use existing recursive count checks. Bare collection assignment, standalone enum codecs, subsequent mutation, and requestBuilder replacement are not schema validation barriers.

With validation, [bounded nested String enum bounds](nested-string-enum-bounds.md) separately support minLength/maxLength/pattern and existing lexical metadata on ordinary direct inline tree leaves; numeric bounds remain rejected. Direct plain String enum lists separately support [serialName uniqueness](string-enum-unique.md), including existing nested paths and defaults. Lists whose elements are finite typed List/Map trees use the [structural uniqueness](structural-unique.md) contract. `uniqueItems:false` retains only the effect-free no-op grammar for otherwise supported arrays under validation. Existing builtin6/temporal literal, check, and uniqueness boundaries remain unchanged. Constrained schemas without validation remain rejected; all NoDrop checks still run.

Resource limits remain unchanged: 8 MiB input, source depth 48, 16 checked collection edges, 16,384 units of analysis work, 240 raw enum members, and 240 serialized constructor slots. Before canonical allocation, each original default independently preflights the entire tree of 16,384 JSON values (including containers/null) and 32 KiB compact JSON; canonical JSON has a separate 32 KiB cap. Keys and scalar contents count in bytes, and default value traversal is also limited to depth 48. Required/Patch/discarded annotations count against the same budgets. Limits are reported at the original Pointer; generation failure returns no partial artifacts, name-lock state, or warnings, and native output preserves old owned and handwritten files under the existing transaction. Individual width/depth/byte boundaries do not guarantee maximum width × maximum depth, compiler heap, or platforms.

## Actual types, optimization, and compatibility

The initializer helper binds the actual lexical enum owner and final TypeRef after optimization or relocation. The original leaf declaration remains the source of independent references; no fake enum owner is created and classifiers are not generated through string substitution. Dedup, merge, hoist, and [name-lock](name-lock.md) use the actual type/nullability/check tree, original vocabulary, and effective canonical default identity. Reordering Map keys at each layer preserves output and lock identity; changing the effective default, nullable layer, retained counts, or actual enum String vocabulary/order cannot reuse a different frozen shape. Required/Patch annotations that produce no initializer add no runtime default identity.

This increment admits direct nested enum field defaults that previously returned `E_SCHEMA_UNSUPPORTED`. This contract covers only direct inline fields; admission of the same nested enum tree shape through an ordinary direct `$ref` field's own default is separately granted by [referenced String enum collection field defaults](referenced-string-enum-collection-defaults.md), without changing this page's scope exclusions. Existing default-free nested enum, single-layer default, and builtin6/temporal control Kotlin APIs, wire behavior, output, and lock format remain unchanged. Consumers of new inputs must regenerate and compile. The scope relates to M01/M03/M04/M07/M08, existing M12/V05, and typed A03/A05/A08 consumers; general D03, other default/constraint roles, platforms, and release acceptance remain open.

## Verification entry points and evidence

The feature is named `nested-string-enum-collection-defaults`. The [synthetic fixture](../tests/fixtures/http/nested-string-enum-collection-defaults.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedStringEnumCollectionDefaultsContracts.kt) define consumer inputs and assertions. Core entry points are [typed trees](../core/nested_string_enum_collection_defaults_test.go), [grammar/guards/resources](../core/nested_string_enum_collection_defaults_guards_test.go), and [actual owner/optimizer/identity](../core/nested_string_enum_collection_defaults_identity_test.go). [Native process tests](../internal/cli/nested_string_enum_collection_defaults_process_test.go) check core/CLI parity, read-only behavior, and preservation on failure. The verification matrix covers four trees, Data/Plain × validation off/on, original member Pointers, required/Patch/discarded validation, empty/null/nullable grammar, opaque keys, per-layer canonicalization, resources, and frozen-lock identity. The minimal empty nested String enum input remains a verbatim positive case; other builtin/temporal rejection boundaries remain intact.

The dedicated route is `bash scripts/test.sh feature nested-string-enum-collection-defaults`. Its [feature script](../scripts/kotlin/features/nested-string-enum-collection-defaults.sh) generates fresh Data/Plain × validation projects, compiles and runs checks for actual enum classifiers/descriptors, missing/null/default/unknown under four Json settings, fresh trees, constructor/copy/serialization counts, optimization/frozen locks, typed HTTP, and fragmented live SSE lifecycle. This contract also participates in the `models` and full `http` gates.

Source/default depth budgets bound generator processing work; deep lexical enum binary names can still exceed a host directory's single-filename limit. The feature's source48 probe compiles to a jar and has the original JavaExec load that same jar, retaining the complete classifier and original tree depth. This route's output configuration does not establish acceptance for other directory output configurations, Android/KMP, minimum dependencies, or all resource combinations.

`OPENAPI_KT_PREPARE_ONLY=1` performs only native generation/check and consumer project preparation; it cannot count as a Kotlin compilation/runtime PASS. One bounded feature pass does not close all of D03 or new release candidate gates.
