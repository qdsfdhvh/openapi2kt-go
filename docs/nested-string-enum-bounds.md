# String enum bounds in finite List/Map fields

With `--validation`, ordinary DTO fields accept finite direct inline List/typed
Map paths ending in lexical String enums with `minLength`, `maxLength` and the
existing portable `pattern`. List/List, List/Map, Map/List, Map/Map and longer
mixed paths use the same original field planner, with or without an outer field
default. The existing [single-layer](string-enum-collection-bounds.md) and
[scalar](string-enum-field-bounds.md) planners retain their established output.

```json
{"properties":{"trees":{"type":"array","nullable":true,
 "items":{"type":"object","nullable":true,"additionalProperties":{
   "type":"string","nullable":true,"format":"date",
   "enum":["one","two","x"],"minLength":2,"maxLength":6,
   "pattern":"^[a-z]+$"}},"default":[null,{"a/~":"one","nil":null}]}}}
```

The getter uses the actual lexical enum, for example
`List<Map<String, TreesInnerValue?>?>?`. Supported format/native metadata remains
lexical and adds no date, UUID, byte-array or provider dependency. Metadata does
not add null vocabulary members or nullable inference; plain String enums keep
their existing [null-member grammar](string-enum-null-members.md).

## Values, defaults and presence

Constructors, Data copy and automatic JSON decoding check each non-null leaf's
actual `serialName`. Known `X` remains an enum entry but fails this field's
minLength. `Unrecognized("future")` passes these checks. Independent enum lookup
and codecs retain open runtime vocabulary. String length counts Unicode
codepoints; patterns use the existing portable substring-search profile.

Each List, Map and enum layer retains independent nullability. Null skips only
its own subtree. Container counts use actual sizes; structural `uniqueItems`
uses the existing [typed tree equality](structural-unique.md), including enum
serialName, map key-order independence and at most one equal null element.
Plain enum defaults remain strict members of the original vocabulary in
addition to satisfying every applicable check. Valid runtime unknown words
remain invalid default annotations. Default pattern matching keeps its existing
printable-ASCII policy.

Every original schema node validates before empty/null default shortcuts.
Malformed vocabularies, metadata and constraints report their original schema
Pointer. Invalid annotation leaves report the consuming `/default/...` Pointer,
including escaped map keys. Keys such as `$ref` remain opaque data. Successful
maps sort keys for deterministic emission after source-order validation.

Required fields validate annotations before suppressing the initializer.
Optional explicitly nullable request fields retain Patch.Absent, Null and Value
with bare JSON wire. Initializers decode fresh trees with the final actual enum
serializer. Existing ordinary object defaults, plain DTO component/alias
factories and consuming DTO collection defaults may include these fields under
their [independent annotation contract](bounded-enum-object-defaults.md).
Annotations are not inherited between owners.

Typed DTO HTTP responses and known SSE payloads run the same checks. Invalid
fragmented live SSE data fails and closes its input; the caller's client and a
new cold stream remain reusable. Caller collection mutation, standalone
collection codecs and request-builder replacement keep their existing limits.

## Original sources and boundaries

Permission belongs to the exact original inline enum terminal. A same-byte copy
or unrelated leaf cannot acquire it. Actual source, declaration, nullable type,
checks, vocabulary and effective default participate in optimization and naming
lock identities. Equivalent trees bind the actual canonical enum owner after
deduplication, hoisting or merging. Lock formats remain unchanged.

Ordinary shared/nested DTOs and ordinary inline request/response/SSE DTOs use
this field contract. Eligible ordinary allOf constructors may retain an original
field, without new cross-contribution aggregation. [Checked enum roots](checked-string-enum-roots.md),
referenced/borrowed enum leaves, child-owned
defaults, tuple/const/value-class sources, union/hierarchy fields and discarded
or raw projections keep their separate eligibility. Validation-off rejects the
original constraint keyword instead of dropping checks.

Original limits remain: input8MiB, source48, work/value16384, default32KiB,
enum240, checked collection edges16 and JVM constructor slots240. No limit or
compiler heap is raised. Independent shallow240-field and single-field depth16
compilation probes do not establish Cartesian width/depth, other platforms or
minimum dependency support.

**Compatibility:** eligible formerly rejected nested fields now generate;
existing accepted API/wire and persisted formats retain their contracts.
Regenerate and compile new consumers. Invalid constructed/decoded nested values
throw at the existing DTO boundary. This extends D03/D05/M03/M04/M07/M08/M12/V05
and typed HTTP/SSE consumers, leaving general graphs and final acceptance open.

[Core values and scopes](../core/nested_string_enum_bounds_test.go),
[guards and budgets](../core/nested_string_enum_bounds_guards_test.go),
[bounded fuzz](../core/nested_string_enum_bounds_fuzz_test.go),
[original tree permission](../internal/schema/nested_string_enum_bounds_test.go)
and [native failure preservation](../internal/cli/nested_string_enum_bounds_process_test.go)
define the profile. The existing `string-enum-collection-bounds` fixture, consumer
and feature route include nested constructors/copy, four Json settings,
required/Patch, factories/object defaults, counts/uniqueness, actual optimization
owners and HTTP/fragmented SSE. Prepare-only is generation/check evidence;
Kotlin compilation and runtime evidence require executing the feature route.

[Collection roots and aliases](bounded-enum-collection-roots.md) now have a separate bounded lexical enum contract, including independent factories, consuming defaults and direct HTTP/SSE checks.

[Checked lexical enum borrowing](borrowed-checked-string-enums.md) separately admits already retained ordinary field/collection-root sources through existing references, with the original enforcing checks and independent eligible defaults. Standalone checked enum components and other source roles keep their own eligibility.
