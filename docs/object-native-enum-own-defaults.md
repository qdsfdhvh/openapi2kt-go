# Native enum own defaults inside object defaults

An ordinary object's numeric/Boolean enum property can retain its existing own
`default` while the surrounding ordinary object/allOf default walk validates
an outer annotation. Existing Boolean, Int, Long, Float and Double backings are
preserved. This includes eligible direct component factories, nested ordinary
objects and independently defaulted finite component aliases.

For example, a Boolean property with `enum:[true,false]` and `default:false`
keeps its constructor default when an outer factory decodes `{}`. A target's
root `default:{"flag":true}` belongs only to that target's factory; an alias
with its own `{}` receives the actual field default false. A nullable alias can
own null, and a nearer alias can narrow only the root. Referencing fields do not
inherit any component factory. Required/Patch behavior retains its existing
contract; owning a default does not satisfy a required member in an outer literal.

Every own annotation and supplied outer value matches the original native
vocabulary strictly, including before Float/Double rounding. A value that rounds
to a permitted Float is still rejected when its original number is absent from
the vocabulary. Int/Long limits, explicit nullability, source formats and
validation-gated bounds/domain checks reuse the existing field planners. Runtime
validation remains opt-in, and the native-enum approximation warning retains its
existing code and source location. No nominal enum wrapper or new wire type is
introduced.

The walk borrows original source nodes/Pointers and retained scalar/check types.
Each native vocabulary in this own-property role is bounded to 240 members in
both validation modes, including for null/empty enclosing defaults. Source48,
shared object work16384, value16384/default32KiB, checked16 and JVMslots240 remain.
Collection elements do not acquire an own-default role here. Ordinary object-union
walks reuse their existing constructor-property admission; Cartesian selected
constructors have the separate [composed member contract](composed-member-defaults.md).
Root/wrapper/target annotation roles outside these contracts remain separate.

The existing `reference-wrapper-component-defaults` route checks fresh
Data/Plain × validation-off/on profiles, four JSON configurations, baseline and
combined optimized/locked output, all five scalar backings, extrema, nullable
aliases, fresh instances, actual field decoding defaults and no inheritance.
Independent 240-property and 240-vocabulary consumers compile and execute.
Native CLI tests compare core bytes, check successful JSON reports and no-op/readonly behavior
and old-output preservation for invalid own and outer Float annotations.
The complete HTTP registry invokes the same consumer. This does not establish
additional platforms or general default graphs.

Previously unsupported ordinary native-enum own-property factory inputs can now
be generated. Existing accepted Kotlin API/wire and manifest/name-lock formats
retain their contracts; no migration is needed for previously valid output.
