# Outer allOf defaults with retained UUID checks

With validation enabled, an ordinary allOf constructor's outer object default
can include repeated direct `string/uuid` fields whose complete original
contributions already qualify for the shared primitive constraint merge. Every
contribution must retain its original source, format and effective nullability;
only groups with actual retained discarded checks qualify. This extends the
existing [String](allof-string-defaults.md) and
[numeric](allof-numeric-defaults.md) default roles using the actual Uuid type.

The merged length interval and admitted single portable pattern apply to the
UUID's canonical lowercase, hyphenated text. Uppercase and compact input
spellings are parsed through the existing UUID rules before these checks; a
compact 32-character token can satisfy a canonical length of 36. Invalid UUIDs
and values outside retained constraints fail at their original value Pointer.
The emitted initializer uses the existing Uuid literal/serializer; it does not
construct a String property or introduce a new codec.

Every original contribution and its own annotation validates before override.
A discarded own default must satisfy its own source. A winning own default and
an outer supplied member must satisfy all retained checks. Omitting an outer
member uses the actual constructor's initializer. Required/Patch suppression
still happens after source validation; root/field/member/container and
independent alias factories retain their own annotation/no-inheritance rules.
The ordinary bounded object walk handles existing nested/list/map/member roles
and nullable layers without granting new reference or ownership permission.

The planner retains original nodes, Pointers, actual constructor/codec types and
shared source/work/value/default-byte/checked/JVM-slot budgets. Discarded fields
are charged even for empty, null or suppressed annotations. Literal-only
aggregate plans do not change emission hints, schema identity or persisted
locks. Missing source authority, unchecked duplicate groups, enum/native/ref
property contributions, other formats and general composition remain separate.

`bash scripts/test.sh feature allof-uuid-defaults` generates fresh Data/Plain
validation projects with baseline and combined dedup/hoist/merge/locked output.
The [fixture](../tests/fixtures/http/allof-uuid-defaults.json) and
[consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfUUIDDefaultsContracts.kt)
exercise four Json configurations, original/winning/outer factories, independent
nullable/narrowed aliases, nested/list/map defaults, required/Patch, actual typed
constructor/serde failures, 240 repeated checked fields, HTTP and fragmented SSE
failure/closure/reuse. Go tests check canonical uppercase/compact values,
invalid own/winning/outer annotations with original Pointers, AST/source identity
and the shared work boundary. Native tests compare core bytes, readonly/no-op
behavior and failure preservation; bounded fuzz checks conjunction/nullability
and deterministic output. Other platforms and broader defaults/aggregation
require independent evidence.

**Migration:** formerly rejected eligible UUID outer-default inputs now produce
the existing Uuid Kotlin API and canonical UUID JSON wire. Existing accepted
inputs, constraint semantics, manifests and name-lock formats retain their
contracts. The earlier [UUID migration](nullable-uuid.md) remains separate.
