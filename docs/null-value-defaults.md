# JsonNull members and finite collection defaults

Admitted ordinary object-default walks accept direct `type: null` and
null-only `enum: [null]` properties. The original schema node and vocabulary
remain authoritative; no enum declaration, type node or serializer is invented.
Existing inline/named/allOf objects, finite inline object trees and their
component/wrapper factories retain their actual constructor and codec binding.

A required member must occur in the annotation. An omitted optional member
retains its own admitted default or neutral initializer. Supplied values must
be actual JSON null; String "null", numbers, Booleans and containers fail at the
original consuming default member Pointer. Required enclosing fields validate
annotations before suppressing initializers, and optional request fields retain
Patch.Absent. Component annotations never become consumer field defaults.

Finite inline List/typed Map default trees also admit these direct JsonNull
leaves, including a single List or Map layer. Nullable intent remains local:
`List<JsonNull>` decodes JSON null entries as JsonNull, while
`List<JsonNull?>` decodes them as Kotlin null through the existing nullable
codec. A nullable leaf grants no enclosing container nullability. Empty or null
annotations still validate every original schema node and enum vocabulary.

The same collection grammar admits component own-default factories and eligible
pure-reference wrappers. For example, `Rows` with `type: array`,
`items: {type: null}` and `default: [null]` emits
`typealias RowsDto = List<JsonNull>` and `fun `RowsDto default`(): RowsDto`.
Factories decode a fresh tree per call. A wrapper owns its independent default;
referencing fields retain neutral/required/Patch behavior. Standalone nonnullable JsonNull
scalar component default factories use the separate
[own-default contract](null-component-factories.md).

Element defaults, const elements, new reference/composition leaves, raw JSON,
uniqueItems on JsonNull arrays and general default graphs retain separate
boundaries. This does not expand global scalar uniqueness or pin equality.
Original source48/work16,384, whole-value16,384 nodes/32KiB, checked16,
JVMslots240 and enum240 limits, declaration/ownership/name-lock identities and
failure preservation remain unchanged. Nulls and map keys still consume value
work and bytes. The helper key uses a bounded scalar token, not a qualified
classifier name.

`bash scripts/test.sh feature null-value-defaults` freshly generates Data/Plain
with validation off/on, baseline and combined optimized/locked modes. Runtime
assertions cover four Json settings, inline/named/allOf/finite object defaults,
plain/null-enum List/Map leaves, independent root/wrapper factories, fresh
instances, required suppression, no inheritance, Patch, HTTP and fragmented SSE
failure/closure/client reuse. A separate input compiles240 List<JsonNull> fields
with their own defaults. Core/source tests check original node/type/vocabulary
identity, exact value budgets and failure Pointers. Native CLI tests cover
core parity, readonly/no-op and invalid field/object/collection/wrapper
annotations preserving old output. Wider platform and final candidate gates
remain separate.

**Migration:** eligible annotations previously rejected now generate. Existing
accepted Kotlin APIs, JsonNull/null codecs, wire behavior and persisted formats
retain their contracts.

[DTO JsonNull collection getter uniqueness](dto-null-collection-unique.md) separately admits finite inline List/typed Map getters with original direct plain/null-only enum leaves. It retains actual getter/initializer/source authority and existing nullable codec collapse; global structural uniqueness remains separate.
