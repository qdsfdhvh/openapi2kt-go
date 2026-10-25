# Factories for unformatted numeric enum components

A numeric enum component may own `default` without an explicit `format`.
The existing native mapping remains Int for `type: integer` and Double for
`type: number`; explicit int32/int64/float/double formats retain their contracts.
No new wrapper, scalar inference or wire type is introduced.

```json
{"Code":{"type":"integer","enum":[1,2],"default":1},
 "Ratio":{"type":"number","enum":[1.5,2.5],"default":1.5}}
```

Factories use the existing top-level typealias form, for example
`` `CodeDto default`() ``. Existing eligible component aliases can own independent
annotations. Referencing fields and required parameters do not inherit root
factories, and request-only nullable fields keep Patch.Absent.

Every annotation strictly matches the original vocabulary and mapped scalar kind
with validation on or off, including before Double rounding. An unformatted
integer remains within Int range and rejects a fractional/exponent integer token;
it does not implicitly widen to Long. Double keeps its existing finite IEEE
mapping, including negative zero. A rounded-equivalent number absent from the
original vocabulary still fails at `/default`. Runtime native membership remains
opt-in in typed consumers; standalone alias assignment adds no validation barrier.
The existing approximation warning and scalar serializer retain their policies.

Explicit nullable native roots and null vocabulary members keep their existing
validation-on admission, nearest alias nullable intent and raw member counts;
no vocabulary infers nullability. Required/Patch/source constraints still validate
before suppression. Original schema nodes/Pointers authorize the root annotation;
a same-byte copied source receives no permission. Source48/work16384,
value16384/default32KiB, native vocabulary240, checked16 and JVMslots240 remain.
Manifest and name-lock formats do not change.

`bash scripts/test.sh feature unformatted-native-factories` checks Data/Plain ×
validation-off/on with baseline and combined optimized/locked output. The
[fixture](../tests/fixtures/http/unformatted-native-factories.json) and consumer
exercise four Json configurations, Int extrema, Double rounding and negative
zero, independent alias factories, vocabulary240, actual constructor/default
presence, no inheritance, required/Patch and controlled HTTP/live UTF-8 SSE
failure/closure/client reuse. A 120-required-Double consumer reaches JVMslots240;
validation-on profiles also compile/run explicit nullable and 240-member null
vocabularies. Native tests compare core bytes and readonly/no-op output, then
preserve old output on invalid direct/alias/raw-vocabulary values. Source tests
and bounded alias fuzz retain original permissions, strict membership and
resource limits. Preparation-only mode is separate from Kotlin runtime evidence;
additional platforms require their own checks.

**Migration:** formerly rejected unformatted numeric enum root annotations now
generate factories using their existing Int/Double aliases. Accepted inputs,
call-site types, wire behavior and persisted formats retain their contracts.
Other root/default/enum compositions and general D03/D04 remain open.
