# Component factories for DTO collections

An eligible List/typed Map component can own a root `default` whose finite inline
container tree ends in ordinary DTO objects. Supported leaves are inline ordinary
objects, a direct named plain DTO reference, or a bare ordinary allOf constructor
already admitted by the object-default walk. The factory returns the actual root
alias using its final serializer.

```json
{"Rows":{"type":"array","items":{"properties":{"value":{"type":"string"}},
 "required":["value"]},"default":[{"value":"first"}]}}
```

```kotlin
import org.example.rows.models.`RowsDto default`
val rows = `RowsDto default`()
```

Typealiases cannot own companion members, so the factory is a top-level function.
Each call creates a fresh typed tree. Actual constructor property defaults are
used for omitted optional members; required members must appear. String/native
enum own-property defaults retain their original strict annotation vocabulary
and runtime policy. Native numeric values keep their existing mapped precision.
The root annotation is opt-in: referencing fields and HTTP arguments do not
inherit it. A consuming required field stays required, and request-only Patch
fields keep Absent/Null/Value.

Existing eligible wrapper aliases may declare independent factory annotations.
The target's annotation is not inherited. Nearest nullable intent changes only
the root; nullable container layers and DTO elements retain independent intent.
Null or empty annotations still validate complete source trees. Underlying typed
references/declarations must remain admitted by the existing resolver and final
nominal binding; the factory creates no temporary DTO declarations.

Qualification initially marks the structural source role, before catalog
construction. The real type plan and complete original-node default walk then
validate every source and value. Failed probe permissions are isolated, and
excluded root annotations cannot be silently accepted without a factory.
Bare allOf leaves receive only their original local constructor permission;
const-only/raw/composed-union/hierarchy roles remain separate. Source/own errors
retain original schema Pointers; invalid supplied factory/alias members use the
corresponding `/default/...` path. Map keys canonicalize only after validation.

The inline container path does not add inter-container reference grammar.
Element alias/internal/allOf references, primitive/object-union or hierarchy
factories, target/element-root/child-container default annotations and other
excluded root/count roles remain independent. Source48, shared work16384,
value16384/default32KiB, checked16 and JVMslots240 remain unchanged.

`bash scripts/test.sh feature object-collection-factories` freshly generates
Data/Plain × validation-off/on, baseline and combined optimized/locked output.
The [fixture](../tests/fixtures/http/object-collection-factories.json) and consumer
cover four Json configurations, direct/named/allOf/mixed leaves, actual field
own defaults, nullable roots/elements/aliases, fresh values, required/Patch and
no inheritance. Independent 240-property and 15-container consumers compile and
run. Controlled HTTP and byte-fragmented UTF-8 live SSE exercise actual typed
lists, callbacks, required-field failure/closure and client reuse. Native tests
compare core bytes and successful reports, readonly/no-op behavior and old-output
preservation. Original-node/probe/shared-work tests, byte/slot boundaries and
bounded fuzz verify source and literal limits. Preparation-only mode does not
claim Kotlin execution; other platforms need separate evidence.

**Migration:** formerly rejected eligible DTO collection root annotations now
generate explicit factories. Existing accepted Kotlin API/wire behavior and
manifest/name-lock formats retain their contracts. General D03 and wider
collection/source/default roles remain open.
