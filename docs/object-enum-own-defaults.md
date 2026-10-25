# String enum own defaults inside object defaults

An ordinary object property's existing String/typeless enum `default` can
participate in the surrounding object-default walk. This includes ordinary
inline objects, direct component factories and eligible ordinary allOf
constructors and their independently defaulted component aliases. For example:

```json
{"Target":{"properties":{"mode":{"type":"string","enum":["on","off"],"default":"on"}},"default":{"mode":"off"}},
 "Nullable":{"$ref":"#/components/schemas/Target","nullable":true,"default":null},
 "Narrow":{"$ref":"#/components/schemas/Nullable","nullable":false,"default":{}}}
```

The target's opt-in factory produces `mode=off`; the nullable alias factory
returns null; the narrowed alias factory decodes `{}` and receives the actual
constructor field default `mode=on`. The target's root annotation is not
inherited. A field referencing any of these components still keeps its own
presence/default contract. Required fields retain their constructor/decoding
requirements, and the general Patch contract does not change.

Each property's annotation is checked strictly against its original vocabulary,
even when an enclosing annotation is null, empty or omits the property. Supplied
outer values are also checked against that vocabulary; runtime unknown Strings
remain open enum values. No new format/bounds/native/const or collection-element
annotation role is granted. Ordinary tag/presence object-union walks reuse this
ordinary constructor role; Cartesian selected constructors use the separate
[composed member contract](composed-member-defaults.md).

Structured alias factories now borrow the original target and child nodes,
retaining source order and Pointers. Only the root validation plan receives
nearest alias nullable intent; children are not copied, narrowed or expanded.
Bad schema/own-property annotations report their original source path. Bad alias
values report the alias's `/default/member` path. Source48/work16384 and existing
value16384/32KiB/checked16/JVMslots240 caps remain. Scalar/raw alias factories
retain their existing annotation-view and grammar policy.

This corrects previously rejected ordinary property-own enum defaults in the
surrounding default walk. Existing Kotlin names, constructors, serializers,
wire behavior and persisted formats retain their contracts. Structured source
diagnostics previously synthesized under an alias path now use original target
paths, and warnings are deduplicated by original source. Diagnostic consumers
must follow those original paths. Run
`bash scripts/test.sh feature reference-wrapper-component-defaults` for fresh
Data/Plain × validation profiles, four Json configurations and combined
optimized/locked output; the complete HTTP registry invokes the same consumer.
This increment does not establish general default graphs or additional platforms.

Numeric/Boolean properties have the separate [native own-default contract](object-native-enum-own-defaults.md), preserving scalar backings, strict pre-rounding annotation membership and existing validation policy.
