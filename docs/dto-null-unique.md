# DTO JsonNull getter uniqueness

The [ordinary DTO uniqueItems profile](dto-unique.md) accepts direct JsonNull
constructor fields and null const-body getters. Plain type:null, explicit or
inferred null-only enum sources and explicit/inferred null const sources keep
their existing Kotlin types, codecs and original nodes. Validation is required;
no enum declaration, synthetic type key or arbitrary JSON equality permission
is introduced.

Keys read actual getters. Required non-null JsonNull arguments and fixed null
bodies retain JsonNull; optional/nullable constructor fields can hold Kotlin
null. A supplied JsonNull object and Kotlin null are distinct mapped getter
values. Their existing nullable codec can encode both as JSON null and decode
wire null as Kotlin null. This wire collapse is unchanged and is not reversed
by uniqueness checks. Json encodeDefaults/explicitNulls do not replace actual
getter values with serialized JSON.

Null-only field default:null retains its existing optional null initializer,
required initializer suppression and vocabulary-first validation. Null const
bodies keep their existing non-null always-encoded body properties, with no
constructor/copy/Patch parameter. Invalid String "null" defaults, duplicate enum
members and wrong-value codec inputs retain their original diagnostics. No
component/child annotation is inherited, and direct null-const object-default members use the [object-default profile](null-const-object-defaults.md).

Admission remains local to the actual retained DTO. Original source/member/
Pointer and exact actual getter/body types, names and initializer plans must
match. Raw JsonElement/JSON projections, foreign body codecs, copied nodes and
same-byte null vocabulary replacements cannot authorize this role. Inferred
sources gain no type node or String enum declaration. Null initializers retain
the null default-key state. General nested/field-reference/composed/hierarchy/
projected/count/const-only DTO roles stay separate.

Source48/checked16/serialized-JVMslots240, enum240, shared capture16,384work/
8MiB, default-key16,384work and other value/width limits remain. The local
JsonNull getter rule does not expand global scalar/structural/raw collection
admission or alter descriptor/source identity protocols.

`bash scripts/test.sh feature dto-null-unique` checks fresh Data/Plain baseline
and validation, combined optimized/locked output, named/inline/nullable DTOs,
required/null-only defaults, JsonNull versus Kotlin null, invalid codec values,
other scalar/enum/const getters, Patch, HTTP duplicate responses and fragmented
SSE failure/closure/client reuse. A wide profile retains240 serialized fields
including a JsonNull body and240 referring collections. Source tests check
original null vocabulary/type absence, actual getter/body types, default keys
and raw projection refusal. Native CLI and bounded deterministic fuzz separately
check original grammar/default errors and failure preservation. This profile
has no additional platform or full graph acceptance claim.

**Migration:** eligible DTO uniqueItems inputs with direct JsonNull fields or
null const bodies formerly rejected now generate under validation. Existing
accepted APIs, null codecs/presence, wire behavior and manifest/name-lock
formats remain.

[DTO JsonNull collection getter uniqueness](dto-null-collection-unique.md) separately admits finite inline List/typed Map getters with original direct plain/null-only enum leaves. It retains actual getter/initializer/source authority and existing nullable codec collapse; global structural uniqueness remains separate.
