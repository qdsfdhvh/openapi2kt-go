# Native enum checks in typed consumers

`--validation` extends the [mapped native enum membership contract](native-enum-validation.md) beyond direct DTO fields. Plain integer/number/Boolean component definitions retain typed membership metadata through existing named scalar aliases/references. Direct native enum schemas inside typed List/Map leaves, HTTP scalar values and parameters, and SSE typed data retain the same checks. Default-off Kotlin files/wire types and original warning policy remain unchanged.

The public declarations remain native scalar/typealiases, not wrappers. A `Code` enum `[1,2]` still exposes Kotlin Int. Assigning `3` to a standalone alias or decoding it through the alias serializer alone does not enforce membership. Checks run in actual typed consumers: a generated DTO constructor/decoder, a containing typed List/Map use, operation input preflight/output decoding, or generated SSE event construction. Existing named DTO constructors remain nominal validation barriers.

## Consumer behavior

| Context | Enforcement |
|---|---|
| DTO field using a native enum component/finite named aliases | Constructor/decode checks mapped value; nullable/optional/Patch presence retained |
| Direct native enum List items or typed Map values | Non-null leaves checked through existing recursive typed consumer traversal |
| Transparent collection aliases | Consumers check their linked typed backing; independent alias values stay unchecked |
| HTTP path/query/header scalar, native enum list/map values | Input preflight before engine request; parameter wire encoding/style behavior unchanged |
| HTTP JSON scalar/collection body | Preflight over typed input, with existing requestBuilder override boundary |
| HTTP JSON scalar/collection response | Checks actual decoded result before returning it |
| SSE scalar/collection data | Generated event constructor checks decoded data with existing transport ownership/cancellation behavior |

Native enum roots are fully grammar/literal validated even when unused. Enforced plain sources stop reporting `W_ENUM_NATIVE_APPROXIMATION`; transparent declarations remain unable to police independent alias assignment. The CLI warning policy sees the updated warning list. Defaults remain direct DTO annotations only; enum roots/ref wrappers/element or value schemas do not acquire default admission. Direct numeric/Boolean enum List/Map field defaults follow their [separate contract](native-enum-collection-defaults.md). [直接嵌套 native enum own-default](nested-native-enum-collection-defaults.md)另接受 ordinary direct inline finite List/typed Map 树外层字段自身注解，严格原 default membership 与 mapped supplied membership 分开；不扩大 roots/refs/named/child/default-inheritance 角色。

The same mapped Float/Double semantics, finite-value check and 240 raw-member limit apply. Alias/reference resolution preserves original catalog/node/Pointer and accumulated source hops. It adds no nominal barrier or copied expanded tree. Existing 48 schema/source depth, 16 checked collection edges and 240 serialized/operation slots remain; depth errors point to the retained original check source. Explicit consumer nullable behavior follows the existing scalar reference policy. [Nullable native enum roots](nullable-native-enums.md) separately admit checked intrinsic/own nullable native scalar roots and finite aliases; arbitrary nullable graphs remain separate.

[Null native enum members](native-enum-null-members.md) additionally admit one null in an explicitly nullable numeric/Boolean vocabulary under validation.

[Numeric enum plus bounds](native-enum-bounds.md) additionally retains endpoints and integer factors; floating factors and general aggregation remain outside that profile.

[Internal native enum Pointers](internal-native-enums.md) additionally borrows direct numeric/Boolean leaves and root bridges under validation with original source identity.

[Named native enum borrowing](named-native-enums.md) additionally retains checks from actual named catalog sources through internal/borrowed graphs.

## No-drop and unsupported roles

Ignored additionalProperties, discarded allOf fields and raw/stringification projections cannot erase enforcing enum metadata; validation rejects those source `enum` locations. Generic primitive union pins, multi-branch scalar enum union aggregation, const intersections, native overrides/value-class annotations, String/null-only vocabularies, general internal/named enum graphs and closed String-enum validation remain separate. A single scalar+null fold can reuse the direct scalar check when existing normalization retains that branch; exact oneOf match-count is unchanged.

Plain native enum root planning is narrowly admitted. Special/native annotated enums remain on their original strict path, preserving earlier alias failure locations rather than moving a diagnostic to an eagerly checked target. No schema/default/example payload is rewritten or reinterpreted as code.

## Compatibility

This is an opt-in V05/M08/M12/A03/A05/A08 behavior change: validation-enabled callers passing nonmembers through these previously unchecked typed consumers now fail. For example, a DTO field referring through `ANumber` → `ZNumber` with vocabulary `[1,2]` previously accepted 99 under validation; it now rejects 99 while valid 1/2 round-trips retain scalar types and wire encoding. The original nonmember is covered as a constructor/decode rejection. Valid calls retain their scalar Kotlin types and JSON/parameter encoding. Default-off generation remains byte-compatible. Typed checks participate in merge/dedup/name-lock identity; changing a checked vocabulary cannot reuse a locked old shape. No persisted format or CLI product version changes.

[Core consumers](../core/native_enum_consumers_test.go), [source/depth guards](../core/native_enum_consumers_guards_test.go), [reuse identity](../core/native_enum_consumers_optimization_test.go), [native output preservation](../internal/cli/native_enum_consumers_process_test.go), [original private identity](../internal/schema/native_enum_consumers_test.go), the [synthetic fixture](../tests/fixtures/http/native-enum-consumers.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumConsumerContracts.kt) define the behavior. `bash scripts/test.sh feature native-enum-consumers` generates fresh Data/Plain × validation on/off with four Json settings and alias/List/Map/Patch/query/body/response/SSE assertions. Invalid operation inputs fail before engine calls; invalid decoded results/events fail while later valid requests remain usable. General graph/pin/String validation and actual platform/toolchain acceptance remain separate.
