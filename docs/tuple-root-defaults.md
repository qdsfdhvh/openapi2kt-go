# Scalar tuple component default factories

An OpenAPI 3.1 array component with `prefixItems` can own a default factory when
its selected List element uses the existing direct scalar tuple-field grammar.
A uniform prefix selects its original first member; an explicit admitted scalar
`items` tail owns the List type after every original prefix member validates.
Plain String, Boolean, Int/Long, Float/Double, Uuid, LocalDate and Instant mappings
retain their ordinary codec contracts. Null intent belongs to each source layer.

```json
{"Rows":{"type":"array","prefixItems":[{"type":"integer"},{"type":"integer"}],
         "default":[1,2]},
 "Wrapped":{"$ref":"#/components/schemas/Rows","default":[3]}}
```

The actual component remains a List typealias. Top-level functions such as
`` `RowsDto default`() `` and `` `WrappedDto default`() `` decode each independent
annotation through that alias's serializer. Factories produce fresh values;
referencing fields, required keys and Patch fields do not inherit the root
annotation. A default on a target does not grant a default role to a tuple
position or a nested child. Existing independent alias nullable/false overrides
apply without rewriting target or child schemas.

Every original prefix and tail source validates before an empty/null annotation
is consumed. Supplied members must fit the selected List scalar type and
nullable intent, including temporal lexical/precision checks. Existing
validation-on collection counts and mapped-value uniqueness apply where already
admitted; prefix-only uniqueness retains its separate rejection. The literal
walk borrows original component/selected-member nodes and Pointers, and copied
source trees receive no component permission. Invalid values report their
consuming `/default/<index>` Pointer. Original source depth and tuple work
budgets, value16384/default32KiB, checked16 and JVMslots240 remain.

Mixed raw prefix-only projections, reference/enum/DTO/composed members,
unsupported/Boolean tails and member-own defaults keep their separate factory
roles. A tuple default adds no positional or exact-length validation: generated
Lists retain the existing `W_TUPLE_APPROXIMATION`, scalar codecs and tail policy.
It also adds no runtime validation to standalone List alias assignment.

`bash scripts/test.sh feature tuple-root-defaults` generates Data/Plain with
validation off/on, baseline and combined optimized/locked output. The
[fixture](../tests/fixtures/http/tuple-root-defaults.json) and consumer check four
Json configurations, Long extrema, Float negative zero, temporal/Uuid codecs,
nullable/empty values, independent and narrowed aliases, no inheritance,
required/Patch, controlled HTTP and live fragmented UTF-8 SSE failure/closure/
client reuse. Supplemental consumers compile 240 fields and validation-on
count/uniqueness factories and typed fields. Native tests compare core bytes,
readonly/no-op and failure preservation; source/value tests cover original
identity, excluded sources and exact node/byte limits, with bounded fuzz for
deterministic valid/invalid/null annotations. Additional platforms require
independent evidence.

**Migration:** formerly rejected eligible scalar tuple root annotations now
produce factories. Accepted tuple declarations, source/wire behavior and
persisted manifest/name-lock formats retain their contracts. Wider tuple and
default roles remain separate.
