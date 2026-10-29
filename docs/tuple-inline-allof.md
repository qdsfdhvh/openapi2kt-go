# Inline ordinary allOf positional-array sources

OpenAPI 3.1 `prefixItems` members and schema-valued `items` tails accept direct
ordinary allOf objects that the existing planner retains as real DTO
constructors. This extends D03/D06 and F08/M03/M07/Q05. The source header permits
type/nullable/allOf/properties/required and existing property counts, alongside
ordinary descriptive metadata. Named/internal allOf references keep their
[existing source contract](primitive-tuples.md#referenced-allof-constructor-members-and-tails).

```json
{"Rows":{"type":"array","prefixItems":[
  {"allOf":[{"properties":{"name":{"type":"string"}},"required":["name"]},
            {"properties":{"n":{"type":"integer","default":7}}}]}],
  "default":[{"name":"root"}]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":[]}}
```

The selected element is the normal planner's actual peer or nested DTO, with
its real constructor, fields, checks, defaults, enums and serializer. Existing
allOf property order, later overrides, required-name union and supported bound/
count aggregation remain. This is the existing merged-object approximation,
not a general intersection validator. The actual emitted constructor supplies
`W_ALL_OF_APPROXIMATION` at its original allOf Pointer.

A single inline allOf prefix selects that original source. Distinct original
nodes stay different even with identical fields and project to raw JSON before
optimization. An explicit admitted schema tail wins after every original prefix
validates. Boolean tails retain their existing meanings. `W_TUPLE_APPROXIMATION`
retains lost position/length constraints and possible tail narrowing, including
explicit warnings when false-prefix restrictions are discarded.

Effective element nullability comes from the existing allOf planner: absent a
root override, member null sets intersect. Two nullable ordinary members can
produce a nullable element; a nullable/non-null mixture retains the non-null
view. Container nullability is independent. This does not expand general
nullable or recursive intersection admission.

## Outer defaults and consumers

Outer field, original component-root and independent component-alias own
annotations reuse the existing ordinary allOf object-default walk and the
actual final selected codec. Supported inherited field defaults, original
required/unknown keys, nullable intent, primitive aggregation and JSON property
counts keep their contracts. Wider default graphs and unsupported overlap roles
remain independent; adding a tuple does not make them valid.

In the example, the Rows factory produces an element named `root` with `n = 7`;
the independent Alias annotation produces an empty List. A consuming field
inherits neither annotation. Required annotations validate before initializer
suppression and Patch stays Absent. Factories/initializers decode through the
actual final constructor after lexical linking, relocation or model merging.
Position-owned annotations are not added. Raw projections retain the
[raw tuple default contract](tuple-raw-defaults.md), including opaque JSON data
keys and value/byte limits.

Ordinary DTO fields, nested finite collections, shared JSON HTTP/SSE and existing
directional request/response views retain their normal behavior and limits.
Direct simple-operation tuple arrays also expose their actual operation-owned
allOf element DTO peers. Those peers enforce required keys and ordinary defaults/
checks and use the caller's Json unknown-key policy; they do not use the plain
inline-object Map projection. Operation names reserve peer names independently.
Legacy direct allOf/union operation roots and non-tuple operation projections
retain their separate admission rules.

## Original source and actual declaration

Every original source, including a mixed or tail-discarded allOf member, completes
ordinary analysis and deferred annotation checks in an isolated analyzer. Empty,
const-only, dictionary-only, hierarchy and union-product shapes cannot acquire
constructor permission from a positional slot. Actual constructors must be
nonempty and retain their original schema and Pointer. Original same-byte source
copies, missing/renamed/copied/duplicated constructors, changed element addresses,
missing source facts or nullable/type/check substitutions supply no authority.

The isolated pass exports original source facts and snapshots. Its temporary
classifiers, defaults, warning registrations and emission candidates never enter
the real plan. Selected declarations are planned in their actual model/operation
context and checked against retained physical declarations before emission.
The source pass is shared with inline ordinary DTO tuples and runs once per
document using those source roles; no performance improvement is claimed.

All original allOf branch/reference paths and deferred defaults retain their
Pointers. Overlapping allOf sources share the existing cumulative tuple allOf
capture budget of 16,384 work and 8 MiB terminal bytes across arrays, contexts and
referenced/inline sources. Repeated captures of one original source are reused;
different sources borrowing the same Base still pay their overlapping snapshot
work. The general tuple node/path capture budget, source depth48, default
value16384/32KiB, checked16 and JVM240 limits also remain. The ordinary planner's original contribution heights also remain in collection caches after a raw or
tail projection erases a prefix type. Incoming/forward collection references
still pay those heights, and retained facts reject altered cached heights.
Non-tuple allOf inputs keep their existing budget profile. Independent depth/width probes do not prove
a combined maximum or arbitrary Kotlin/JVM feasibility.

Existing uniqueness eligibility remains; no general nominal/projected equality,
union/product/hierarchy constructor, internal graph or complete JSON Schema
validation role is added. Existing accepted APIs, wire behavior and manifest/
name-lock encodings retain their contracts. Formerly rejected eligible inline
allOf tuples now generate; opt-in merge/hoist retains its existing classifier and
nested-access migration rules. Additional platforms and final candidate
acceptance need separate evidence.

[Core tests](../core/tuple_inline_allof_test.go),
[source/type/constructor guards](../internal/schema/tuple_inline_allof_test.go),
[structured fuzz](../core/tuple_inline_allof_fuzz_test.go),
[native lifecycle](../internal/cli/tuple_inline_allof_process_test.go) and the
[synthetic fixture](../tests/fixtures/http/tuple-inline-allof.json) define this
profile. `bash scripts/test.sh feature tuple-inline-allof` generates/checks and
compiles/runs Data/Plain × validation off/on with baseline and combined optimized
output, actual nonempty hoist locks, 240 independent fields and four Json settings.
The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleInlineAllOfContracts.kt)
covers defaults, nullable/required/Patch, enum/child state, counts/checks, direct
and shared HTTP, directional DTOs and fragmented still-open SSE failure/closure/
reuse. The complete `http` route includes the consumer. PREPARE_ONLY or a source
scan alone does not prove Kotlin execution or release/platform acceptance.
