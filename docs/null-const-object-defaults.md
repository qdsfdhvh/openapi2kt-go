# Null const members in object defaults

Existing ordinary object-default walks accept direct explicit
`{"type":"null","const":null}` and inferred `{"const":null}` properties.
The member borrows the original literal, schema and Pointer; no type node,
enum, class or serializer is invented. This applies where the enclosing object
annotation was already admitted: inline/named DTOs, finite inline containers
and admitted ordinary allOf/component/wrapper factory contexts retain their
existing actual constructor/codec binding.

An omitted optional null const property uses its existing JsonNull body
initializer. A supplied value must be actual JSON null; the String "null",
Boolean/number/object/array values fail at the original consuming default
member Pointer. A required const wire must still be supplied to the annotation.
The field remains a non-null JsonNull body val with EncodeDefault(ALWAYS),
including when its source has nullable:true. It gains no constructor/copy/Patch
parameter or Kotlin-null state. Wrong-value runtime decoding still fails through
the existing JsonNull codec.

An optional enclosing field receives a fresh decoded tree on each construction
or missing-key decode. Required fields validate annotations before initializer
suppression, and optional request fields retain Patch.Absent. Component and
wrapper factories keep independent own values; consuming fields inherit none.
Eligible DTO uniqueItems default keys treat omitted and explicit null const
bodies as the same actual initializer state. Existing source order, original
failure Pointers, actual retained declarations, ownership and lock identities
remain. Null union tag dispatch and global pin/equality profiles are unchanged.

Null members still consume the existing whole-value node/byte budgets. Shared
source48/work16,384, annotation16,384 nodes/32KiB, checked16 and serialized
JVMslots240 remain. Source tests retain literal identity/type absence and check
exact value-work/byte boundaries. Null const+default on the property itself,
structured consts, literal component-root factories and unadmitted references,
container paths or recursive graphs retain separate boundaries.

`bash scripts/test.sh feature null-const-object-defaults` generates Data/Plain
with validation off/on and combined optimized/locked output. It checks
inline/named/allOf/finite tree/const-only defaults, fresh instances, required
suppression, component/wrapper factories, no inheritance, all four Json
settings, Patch, HTTP missing-key defaults, wrong-value responses and fragmented
SSE failure/closure/client reuse. A separate input compiles240 serialized fields
with239 null bodies and one constructor argument. Native CLI checks cover
core bytes, readonly/no-op, original field/factory errors and failure
preservation. Bounded fuzz varies explicit/inferred/nullable sources and
style/options. Broader graph/platform/final candidate evidence remains separate.

**Migration:** eligible enclosing object default annotations containing null
const members formerly rejected now generate. Existing accepted APIs, null
codecs, wire behavior and manifest/name-lock formats retain their contracts.
