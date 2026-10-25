# Ordinary allOf directional HTTP DTOs

Ordinary allOf constructor DTOs can provide operation-owned request and public
response views using the existing [directional DTO contract](directional-models.md).
This includes inline/named allOf contributions and already accepted internal
ordinary DTO operands. Shared component constructors and codecs remain intact.
Request views use Patch for readOnly fields, preserve missing/null/value rules
and required/default suppression; public response views omit writeOnly fields.
Nested DTOs, finite collection aliases and nominal recursion use the existing
operation-owned projection planner.

Field annotations come from the actual winning original field after ordinary
allOf later overrides. Required entries remain the merged member/root set.
An earlier readOnly/writeOnly annotation cannot override a later field's own
annotation. Hidden and discarded source fields are still validated before any
projection; malformed formats or annotations report their original Pointers.
AllOf approximation warnings and existing field/default/check rules remain.
This does not add exact intersection validation.

The model builder retains already validated winning field references and the
merged required set in a private analyzer source index. Directional planning
consumes that index without re-expanding or admitting source graphs after the
final original-declaration check. Source identity/Pointer checks bind the
index to its actual original Model. The index stays outside semantic Model IR,
so declaration equality, optimization and naming-lock identity remain intact.
Directional copies receive no original model/schema authority. Original ASTs
and shared source declarations are not rewritten.

Hierarchy/union/product projection, raw carriers, property-count projection,
readOnly pinned constants and defaults that depend on a changed nominal
constructor retain their existing explicit boundaries. Accepted nullable views
and recursion keep their existing contracts. Source48/work limits, checked16,
JVMslots240, clone-count16,384 and default/value/input limits are unchanged.
A240-field shared/request constructor remains bounded;241 fields reject.

`bash scripts/test.sh feature allof-directional-models` generates fresh
Data/Plain×validation with baseline and combined optimized/locked outputs plus
an independent240-field request. The [fixture](../tests/fixtures/http/allof-directional-models.json)
splits annotations and required entries across original allOf members. Kotlin
consumers exercise four Json settings, actual required call sites, readOnly
Patch, hidden/unknown response fields, unchanged shared DTOs, recursion,
nullable collection aliases, optimized views, HTTP bodies and SSE payloads.
Core/source tests cover named contributions, later overrides, merged required,
discarded/hidden validation and source Pointers, no source authority on copies,
unchanged original AST/capture and constructor bounds. Native tests compare
core bytes, check/no-op and failure preservation; bounded fuzz varies named
contributions, fields, validation and optimizations. Broader graphs/platforms
require separate evidence.

**Migration:** formerly rejected eligible ordinary allOf directional DTOs now
generate using the existing directional call-site and wire rules. Previously
accepted shared DTOs and manifest/name-lock formats keep their contracts.
No automatic error-status dispatch, SSE reconnect or broader schema validation
is added.
