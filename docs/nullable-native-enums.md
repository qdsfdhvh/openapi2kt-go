# Nullable native enum roots and aliases

With `--validation`, a plain numeric/Boolean native enum component may carry intrinsic nullable intent, and finite named root aliases may add or narrow nullable intent. For example `{"type":["integer","null"],"enum":[1,2]}` exports a transparent `Int?` alias. A root reference with `nullable:false` exposes the non-null scalar backing while preserving the vocabulary checks in actual consumers.

This extends [typed native enum consumers](native-enum-consumers.md), [numeric enum bounds](native-enum-bounds.md), [internal enum Pointers](internal-native-enums.md) and [named enum borrowing](named-native-enums.md). It uses dedicated native enum nullable metadata, separate from ordinary builtin, const/native and value-class profiles. Public declarations remain scalar/typealiases. Independent alias assignment or reified alias decoding alone does not enforce membership; generated DTO constructors/decoders and HTTP/SSE typed consumers do.

## Intent, checks and presence

| Context | Contract |
|---|---|
| Intrinsic nullable enum root | Transparent nullable native alias with original mapped vocabulary/bounds metadata |
| Own-nullable/false named alias | The nearest explicit use-site or alias override wins; false uses the checked non-null backing |
| Internal hop/root bridge or borrowed collection | Original named source and actual scalar/nullable export authority; all source work retained |
| Non-null scalar values in typed consumers | Existing mapped membership and numeric constraints are enforced |
| Nullable values | Null is handled by the Kotlin nullable codec and bypasses non-null membership checks |
| Optional request property with inherited nullable intent | Existing `Patch.Absent` / `Patch.Null` / checked `Patch.Value` semantics |
| Nullable HTTP JSON scalar body/result and SSE data | Existing nullable request encoding/decoded scalar projection, then typed checks for non-null values |

[Null native enum members](native-enum-null-members.md) separately admits one null entry in these explicit nullable vocabularies under validation. A non-null vocabulary remains required and an empty admitted non-null numeric-bounds intersection is still rejected, even for a nullable root. Direct DTO defaults keep their separate original-vocabulary/constraint validation; nullable null defaults follow existing behavior. Roots, aliases and collection leaves do not gain defaults. Required nullable DTO fields retain the existing model constructor/decode presence policy; validation does not impose an additional required-key validator.

The original schema node, vocabulary/member nodes, validation object and actual export authorize borrowed graphs. Intrinsic nullable roots require their actual nullable `ValueAlias`; a non-null `ScalarAlias` cannot stand in for it. Narrowed typed views do not mutate that canonical nullable declaration. Original source diagnostics, alias cache heights and checked collection traversal remain. The limits stay 48 source depth, 16 checked collection edges, 16,384 expansion work, 240 raw enum members and 240 serialized/operation slots. Required nullable Long uses one boxed slot; an explicit non-null Long view uses two slots.

## Compatibility and limits

Admission is validation-only. Previously rejected intrinsic nullable/own-nullable native enum root inputs remain rejected without validation. Historical valid numeric/Boolean cases retain their original input bytes and have explicit checked graduation coverage with default-off rejection. Existing non-null supported inputs keep their Kotlin/wire behavior. Nullable behavior participates in merge/dedup and name-lock identity; changing a checked public null view cannot replay an old shape. Persisted format versions are unchanged.

Special native/value-class/const annotations, String/null-only enum vocabularies, general enum composition/intersections, closed String enum checks and primitive union pin enforcement remain separate. Shape borrowing into object allOf/discriminator/primitive union contexts remains excluded for these nullable roots, including aliases narrowed to false. Raw/stringification projections and discarded source fields cannot drop membership or bounds.

[Core nullable behavior](../core/nullable_native_enum_test.go), [source/depth/no-drop guards](../core/nullable_native_enum_guards_test.go), [slot budgets](../core/nullable_native_enum_resources_test.go), [reuse/lock identity](../core/nullable_native_enum_optimization_test.go), [canonical nullable export](../internal/schema/nullable_native_enums_test.go), [native output preservation](../internal/cli/nullable_native_enum_process_test.go), the [synthetic fixture](../tests/fixtures/http/nullable-native-enums.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableNativeEnumContracts.kt) define the contract. `bash scripts/test.sh feature nullable-native-enums` generates fresh Data/Plain profiles under four Json configurations, with alias null decoding, constructor checks, Patch, HTTP null body/results and SSE null data. General graph/enum support and actual platform/toolchain acceptance remain separate.
