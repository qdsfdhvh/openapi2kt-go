# JsonNull scalar component default factories

An existing nonnullable `type: null` or null-only `enum: [null]` component may
own `default: null`. Its actual JsonNull scalar alias gains an opt-in top-level
factory. This is a default ownership role; it introduces no Kotlin enum, type
node, declaration kind or serializer.

```json
{"NullValue":{"type":"null","default":null}}
```

```kotlin
public typealias NullValueDto = kotlinx.serialization.json.JsonNull
public fun `NullValueDto default`(): NullValueDto = /* actual alias JSON decode */
```

The factory returns the JsonNull singleton through the alias serializer. Its
return type is nonnullable. It never returns Kotlin null or manufactures fresh
JsonNull instances. An eligible finite pure component alias may own its own
factory, independently of target/intermediate annotations. An unannotated alias
gets no factory, and consuming fields inherit no annotation. Required fields
keep their constructor arguments; optional/Patch presence and wire behavior
retain their existing contracts.

Only actual JSON null is accepted. String "null", numbers, Booleans and
containers fail at the original owning component/wrapper default Pointer.
Null-only vocabularies validate their original grammar and raw240 limit before
duplicate checks. The analyzer allows an annotation only on the original
component node marked by the factory planner; same-byte clones and unrelated
typed roles retain the strict null-enum grammar. Original nodes, Pointers,
source order, type absence, aliases and warning identities remain authoritative.

Explicitly nullable JsonNull component roots and nullable aliases, const roots,
formats/native/value-class sources, internal/reference/composition graphs
outside the existing wrapper role and general defaults retain separate
boundaries. JsonNull fields, object members and collection defaults retain
their own roles. Shared source48/work16,384, whole-default16,384 nodes/32KiB,
enum240/checked16/JVMslots240, failure preservation, ownership and name-lock
formats remain unchanged.

`bash scripts/test.sh feature null-component-factories` freshly generates
Data/Plain with validation off/on and combined optimized/locked output.
Four Json settings check actual scalar/null-enum/root/intermediate/outer
factories, required/missing/wrong values, no inheritance, existing object and
collection defaults, Patch, HTTP and fragmented SSE failure/closure/reuse.
An independent input retains240 List<JsonNull> fields with own defaults.
Source tests check the original marker/vocabulary/type/declaration authority;
native CLI tests check core parity, readonly/no-op and invalid scalar/wrapper
annotations preserving old output. Bounded alias-chain fuzz verifies options,
determinism and strict owning failure Pointers. Broader platform and final
candidate acceptance remain separate.

**Migration:** eligible nonnullable JsonNull component/alias own annotations
formerly rejected now generate factories. Existing accepted APIs, JsonNull/null
codecs, wire behavior and manifest/name-lock formats retain their contracts.
