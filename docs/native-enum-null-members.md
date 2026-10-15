# Null members in nullable native enums

With `--validation`, a plain numeric/Boolean enum with explicit nullable intent can contain one JSON null member and at least one non-null scalar member. For example `{"type":["integer","null"],"enum":[1,null,2]}` retains an `Int?` backing and checks 1/2 in actual generated consumers. The nullable codec handles null; null never becomes a numeric/Boolean comparison literal. This extends [nullable native roots and aliases](nullable-native-enums.md) without inferring nullability from the vocabulary.

## Vocabulary and consumers

The parser retains the original vocabulary array and every original member node, including null. It validates duplicates and mapped scalar literals at their original indices. A repeated null is rejected at the second null's `/enum/<index>`. Null-only vocabularies, null under a non-null type, native/value-class/const annotations and general enum composition remain outside this profile. [Explicit nullable String null members](string-enum-null-members.md) use a separate open enum contract. An admitted nullable native enum still needs a nonempty non-null numeric-bounds intersection; null does not rescue an empty domain. Floating enum `multipleOf` remains excluded.

| Context | Behavior |
|---|---|
| Direct native DTO field or collection leaf | Nullable codec plus existing non-null membership/bounds checks |
| Intrinsic nullable root, named alias, internal Pointer or borrowed collection | Original source/export authority and nearest explicit nullable override retained |
| Direct DTO default | Non-null defaults match original non-null vocabulary and retained bounds regardless of null position; null defaults retain existing nullable behavior |
| Optional nullable request field | Existing `Patch.Absent`, `Patch.Null`, and checked `Patch.Value` semantics |
| HTTP input/result or SSE data | Existing preflight, decoded result/event checks, nullable scalar body/result and null data behavior |
| Standalone alias assignment or reified alias decoding | Remains unchecked, as with other transparent native scalar aliases |

The original limits remain 240 raw members, 48 schema/source depth, 16 checked collection edges, 16,384 expansion work and 240 serialized/operation slots. Null counts toward the raw member limit: 239 scalar literals plus null fit; 240 scalar literals plus null do not. Required nullable Long consumes one boxed slot; narrowing to non-null consumes two. Raw projections, ignored additionalProperties and discarded source fields cannot drop enforcing checks.

## Compatibility

This admission is validation-only; default-off generation retains its previous rejection. Existing successful inputs and emitted wire types are unchanged. A redundant null vocabulary member in an already nullable schema does not change its non-null check plan or public null view, so dedup/merge and name-lock identity stay stable. Changing a non-null admitted member still changes the checked identity and cannot replay a locked old shape. This does not alter persisted format versions.

Nullability remains the existing generator's nullable intent contract, including its presence policy; this extension does not introduce a stricter required-key or nullable-enum intersection validator. Roots, aliases and collection leaves do not gain default annotations. [直接嵌套 native enum 字段自身默认值](nested-native-enum-collection-defaults.md)仅使用既有叶 null-member grammar；nullable 节点独立、null-member 仍 validation-only，outer-own 注解不授权 child/root/ref defaults。 Broader String/null-only/union/intersection and platform acceptance remain separate.

[Core behavior](../core/native_enum_null_members_test.go), [original diagnostics and budgets](../core/native_enum_null_members_guards_test.go), [source node authority](../internal/schema/native_enum_null_members_test.go), [internal cached source checks](../internal/schema/native_enum_null_internal_test.go), [typed and projected contexts](../core/native_enum_null_members_contexts_test.go), [reuse/name-lock identity](../core/native_enum_null_members_optimization_test.go), [native output preservation](../internal/cli/native_enum_null_members_process_test.go), the [synthetic fixture](../tests/fixtures/http/native-enum-null-members.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumNullMemberContracts.kt) define the contract. `bash scripts/test.sh feature native-enum-null-members` generates fresh Data/Plain profiles and checks missing/null/default/member behavior, Patch and HTTP/SSE under four Json configurations.
