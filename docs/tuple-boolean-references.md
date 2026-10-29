# Named Boolean references in positional arrays

OpenAPI 3.1 `prefixItems` members and schema-valued `items` tails accept references
to existing named literal true/false components and their pure named aliases.
This extends D03/D06 and F08/F10/M03/M07/Q05. The actual existing root declaration
and codec authorize the reference; the general resolver's false-reference gate
and internal/composed/recursive Boolean source roles keep their own contracts.

```json
{"Empty":false,
 "Maybe":{"$ref":"#/components/schemas/Empty","nullable":true,"default":null},
 "Any":true,
 "Zeros":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Empty"}],
          "default":[]},
 "Nulls":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Maybe"}],
          "default":[null]},
 "Raw":{"type":"array","prefixItems":[{"$ref":"#/components/schemas/Empty"}],
        "items":{"$ref":"#/components/schemas/Any"},"default":[null,1]}}
```

`ZerosDto` is a List of the real uninhabited `EmptyDto` class. Its empty List is
valid; every supplied element, including JSON null, fails the rejecting root
codec. `NullsDto` uses the actual nullable root codec and admits null elements
only. The nearest alias/consumer nullable override wins, including explicit false
narrowing. Each false root keeps its own nominal classifier and descriptor.
No positional helper class or absence-carrier field policy is substituted.

Uniform false references reaching the same original root with the same effective
nullability retain that typed List. Aliases to that source may be uniform.
Distinct false roots, direct-literal versus referenced false sources, and mixed
signatures use the ordinary raw List projection. Optimization does not change
this original-source decision or merge opaque false roots.

True references use the existing JsonElement wire carrier. A prefix-only true
projection uses `List<JsonElement>`; an explicit true-reference tail selects
JsonElement with its actual effective nullable view. A non-null JsonElement can
carry JsonNull; a nullable JsonElement codec can decode that wire value to Kotlin
null. The original named true alias remains available separately, with no new
constructor or serializer API.

An explicit admitted schema tail wins after every original prefix validates.
Boolean keyword `items:true` and `items:false` keep their existing tail meanings.
`W_TUPLE_APPROXIMATION` retains lost positions/lengths and possible narrowing; it
explicitly identifies false-prefix restrictions lost by a raw or wider selected
projection. Nullable false tails can lose a non-null false prefix's null refusal.
The named true root retains its own `W_BOOLEAN_SCHEMA_PROJECTION` once at the
original root, independently of the number of tuple consumers.

## Defaults, presence and consumers

Outer field, original component-root and independent component-alias own defaults
reuse the selected codec and original literal validation. Non-null false Lists
permit empty arrays; nullable false elements permit null; true/raw elements keep
original JSON data, including metadata-looking object keys. Container nullability
is independent of element nullability. Existing count/check and uniqueness
eligibility stays in force; this extension adds no general nominal uniqueness.

Required arguments retain no initializer and Patch stays Absent after annotation
validation. Optional initializers and factories decode through the actual final
codec. No target/alias/child default is inherited: the example's Maybe factory
returns null independently of the Nulls factory's explicit List annotation.
Malformed or unsupported discarded references and invalid suppressed annotations
still fail at their original Pointers before output changes.

Ordinary model fields, nested finite collection paths, request Patch, shared and
direct tuple-array HTTP bodies/responses, and typed SSE payloads reuse these
actual codecs. Direct false **operation roots** remain a separate admission
role. Existing response decoding, live SSE failure/cancellation closure, client
reuse, directional fields and optimizer/name-lock behavior retain their contracts.

## Sources, limits and compatibility

Every original reference/alias/root and physical ancestor path is captured,
including prefixes discarded by raw projection or an explicit tail. Same-byte
copied nodes, altered aliases or a missing/duplicated/renamed root declaration
supply no source authority. Retained element addresses and type/check snapshots
bind the selected actual codec; required/default and operation qualification
paths retain that binding.

Original reference hops continue to count after a raw/tail projection erases
the prefix's type. Collection-height caching preserves the maximum original
Boolean-reference prefix work alongside the selected element work, so incoming
and forward collection references cannot turn source depth49 into depth48.
Source depth48, per-tuple reference16384, cumulative tuple capture16384 work/8MiB,
default value16384/32KiB, checked16 and JVM240 limits remain. Cached transient work
is removed before semantic output; public type/wire identity does not acquire it.

Previously rejected eligible inputs now generate. Existing accepted Kotlin
APIs, wire behavior, output ownership and manifest/name-lock encodings retain
their contracts. The source-depth correction applies to the newly admitted
Boolean-reference tuple role. Wider Boolean/internal/default/composed/recursive
graphs and additional platforms remain independent work.

[Core contracts](../core/tuple_boolean_references_test.go),
[original source/type proof](../internal/schema/tuple_boolean_references_test.go),
[bounded fuzz](../core/tuple_boolean_references_fuzz_test.go),
[native lifecycle](../internal/cli/tuple_boolean_references_process_test.go) and the
[synthetic fixture](../tests/fixtures/http/tuple-boolean-references.json) define
this profile. `bash scripts/test.sh feature tuple-boolean-references` compiles and
runs Data/Plain × validation off/on, baseline and combined optimized output with
nonempty hoist locks. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleBooleanReferencesContracts.kt)
checks four Json settings, real rejecting/nullable/raw codecs, field/root/alias
defaults, required/Patch, nested collections, controlled HTTP and fragmented
still-open SSE failure/closure/reuse. Independent 240-field and directional
consumers also compile and run. The complete `http` route includes the main
consumer; these entrypoints do not establish a full option Cartesian product,
additional target platforms or final candidate acceptance.
