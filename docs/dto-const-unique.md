# DTO scalar const-body uniqueness

The [ordinary DTO uniqueItems profile](dto-unique.md) accepts direct builtin
scalar const body fields alongside its ordinary scalar/enum constructor fields.
Validation is required. An actual ordinary constructor must remain; const-only,
structured/caller-native/value-class bodies, reference fields, composed or
projected DTOs and other unproved codec roles keep separate boundaries.
Direct null bodies use the [JsonNull getter profile](dto-null-unique.md).

Scalar const fields keep their existing body val, literal initializer and
EncodeDefault(ALWAYS) behavior. They gain no constructor, copy or Patch
parameter. As documented by the [scalar const contract](model-semantics.md#scalar-const-fields),
decoding can preserve a different mapped body value. Unique keys therefore read
all actual constructor and scalar-body getters. They do not assume const
membership or compare only the literal initializer. Two items with the same id
but different decoded const-body values remain distinct; equal actual getters
are duplicates. Floating getters normalize signed zero and reject non-finite
values. Data/Plain style and Json encodeDefaults/explicitNulls do not change
these keys.

Collection-default annotations retain their original object/const validation.
Omitted body fields use their actual initializer in default keys; an explicit
matching literal can therefore reveal a duplicate at the second original
value Pointer. Invalid const literals retain their original member Pointer.
This does not broaden object-default grammar: format-bearing const fields and
other previously excluded annotation roles retain their independent boundary.
Required/default/Patch presence and no-inheritance rules remain.

Final admission binds the original source/member identities and one retained
DTO to its actual constructor properties and const-body fields. Body name,
wire name, mapped type and initializer must match the original scalar planner;
keys/defaults must match each actual getter. Missing, repeated or changed body
fields and same-byte source replacements are rejected. Source/default metadata
stays private; the enforcing descriptor carries only actual getter/type data.
Original source48/checked16/JVM serialized-slots240, capture16,384work/8MiB,
default-key16,384work and other value/member limits remain.

`bash scripts/test.sh feature dto-const-unique` generates fresh Data/Plain
baseline/validation profiles with combined optimized/locked output, named and
inline DTOs, body decoding/re-encoding, actual getter equality, defaults,
nullable/Patch, HTTP duplicate responses and fragmented SSE failure/closure/
client reuse. Wide inputs retain240 serialized DTO fields, including a scalar
body, and240 referring collection fields. Go source tests mutate actual body
names/types/wires/initializers and original const nodes. Native parity, readonly/
no-op/failure preservation and bounded default/determinism fuzz are separate
checks. Broader const or graph/platform completion is not implied.

**Migration:** eligible DTO uniqueItems inputs with scalar const bodies formerly
rejected now generate under validation. Existing accepted APIs, scalar const
codec approximation, wire behavior and manifest/name-lock formats remain.
