# Bounded lexical enums in object defaults

Existing ordinary object-default walks accept their actual direct String enum
fields and direct List/typed Map enum leaves with retained String bounds.
[Finite nested inline paths](nested-string-enum-bounds.md) use the same checks.
Component factories, direct inline/referenced object fields and DTO-containing
collection defaults use the same original vocabulary, minLength/maxLength and
portable pattern checks. Supported format/native metadata keeps lexical String
codecs. The underlying [field bounds](string-enum-field-bounds.md) still require
validation; they do not become date, UUID, byte-array or provider values.

```json
{"Value":{"properties":{"state":{"type":"string","format":"date",
 "enum":["one","two","x"],"minLength":2}},"required":["state"],
 "default":{"state":"one"}},
 "Alias":{"$ref":"#/components/schemas/Value","default":{"state":"two"}},
 "Holder":{"properties":{"value":{"$ref":"#/components/schemas/Alias",
 "default":{"state":"one"}}}}}
```

Value's companion factory returns state `one`, Alias's top-level factory returns
`two`, and Holder's field annotation independently returns `one`. Unannotated
consumers inherit neither factory. An admitted original ordinary DTO component
or pure named alias may keep its own annotation while a consumer borrows the
shape. This independent-owner rule also applies to existing unbounded ordinary
DTO targets and eligible DTO elements in consuming collection defaults.
It grants no annotation role to an internal, unmarked or copied source node.

Each non-null annotation leaf must be an original enum member and satisfy its
retained bounds. Runtime Unrecognized stays open at the existing constructor
boundaries; a valid unknown runtime word is still an invalid default. Invalid
known words such as `x` fail at the consuming value Pointer. Dictionary values
validate in original key order before canonical sorting; escaped keys retain
their original `/default/...` location. Malformed vocabularies/metadata/checks
retain their original source keyword Pointers, including for empty or null
outer annotations.

Every original field, component, intermediate alias and consumer annotation
validates independently. A valid outer value cannot hide an invalid target or
alias factory. Omitting a field from an object annotation leaves the actual
DTO's own property initializer/required policy in control. Required and Patch
annotations validate before initializer suppression. Optional request objects
retain Patch.Absent/Null/Value and no default update on missing keys. Factories
and object/collection helpers decode through their final actual serializer and
produce fresh structured state. Actual DTO checks remain active on construction,
Data copy, automatic JSON decoding, HTTP and known SSE payloads.

The object walk grants bounds only to the exact original direct enum field or
its bounded terminal in an admitted inline List/Map path.
Checked enum roots/references, child-owned annotations, discarded/ambiguous
declarations, hierarchy/union contributions and cross-contribution aggregation
retain their separate eligibility. Ordinary disjoint allOf factory fields reuse
their existing merged constructor walk; independently annotated allOf/collection
root aliases outside the plain-DTO target profile remain separate. No resolver,
nullable precedence, ownership, persisted lock or runtime vocabulary policy is
broadened. Original schema48, source/value16384, default32KiB, enum240,
checked16 and JVM240 limits remain.

**Compatibility:** previously rejected eligible object annotations and original
plain-DTO target/alias annotations now generate. Existing accepted Kotlin APIs,
wire behavior and manifest/name-lock formats retain their contracts. New checked
annotations must be regenerated and consumed with their actual canonical types;
invalid values fail before output mutation. This extends D03/D05/M01/M03/M04/M07/
M08/M12 and typed HTTP/SSE consumers; generic defaults, complete platforms and
final candidate acceptance remain open.

[Core role/value/ownership/Patch contracts](../core/bounded_enum_object_defaults_test.go),
[bounded fuzz](../core/bounded_enum_object_defaults_fuzz_test.go),
[original source permission tests](../internal/schema/bounded_enum_object_defaults_test.go)
and [native failure preservation](../internal/cli/string_enum_collection_bounds_process_test.go)
define the profile. The existing `string-enum-collection-bounds` feature freshly
generates Data/Plain with validation and unbounded-off controls, four Json
settings, real root/alias/inline/ref/DTO-list defaults, Patch, optimization/locks,
independent240-field probes, HTTP and fragmented SSE failure/close/reuse.


The [five-input platform probe](platform-enum-field-bounds.md) also includes the
original DTO/alias/null factories and consuming object/DTO-list defaults against
these field checks. Its runtime versus compilation and target/style evidence
remain scoped to those exact inputs and fixed toolchains.

[Original lexical enum collection roots/aliases](bounded-enum-collection-roots.md) also retain independent factories inside existing object-default collection graph walks, with final linked-backing validation and no inheritance.
