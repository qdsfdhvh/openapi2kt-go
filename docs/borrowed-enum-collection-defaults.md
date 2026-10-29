# Defaults on borrowed lexical enum collections

An existing finite inline List/typed Map source with lexical String enum leaves
may retain its ordinary field-owned `default` when a component alias borrows that
collection. The borrowing alias and finite pure aliases may each own a separate
opt-in default factory. Plain and metadata vocabularies keep String codecs;
bounded leaves require `--validation` and retain their actual original checks.

```json
{"Owner":{"properties":{"rows":{"type":"array","items":{
 "enum":["one","two","x"],"minLength":2},"default":["one"]}}},
 "Rows":{"$ref":"#/components/schemas/Owner/properties/rows","default":["two"]},
 "Alias":{"$ref":"#/components/schemas/Rows","default":["one"]},
 "Maybe":{"$ref":"#/components/schemas/Alias","nullable":true,"default":null},
 "Use":{"properties":{"value":{"$ref":"#/components/schemas/Alias"}}}}
```

`Owner.rows` initializes to its own `one` value. The Rows factory returns `two`;
Alias returns `one`; Maybe returns null. `Use.value` has no inherited default.
Each non-null factory decodes fresh state through its actual retained alias
serializer, using the original lexical enum classifier. Kotlin factory names
follow the existing [component default convention](component-default-factories.md).

Every annotation validates independently against the complete original source.
Defaults must use the original enum vocabulary, even when runtime unknown words
would pass the String checks. Counts, bounds, uniqueness, nullable layers and
canonical Map keys follow the existing typed default walk. Errors retain the
annotating owner/value Pointer; malformed schema errors retain the original
source keyword Pointer. Empty/null annotations still validate the full source.
Nearest explicit alias intent controls root nullability; it never rewrites the
source's own null grammar. Required/Patch fields keep their existing suppression
and missing/null/value rules.

The source collection retains its original node, ordered parents, graph, linked
backing and check addresses. Factories retain the actual unique alias export,
source and annotation nodes, canonical JSON, binding address, cached alias
height and final TypeRef. A copied binding, duplicate export or changed factory
initializer cannot authorize output. Checked sources additionally retain their
original enum/getter checks under the [checked borrowing contract](borrowed-checked-string-enums.md).

This adds no child-owned annotation, new reference graph or standalone checked
enum-root role. An alias can own a factory for an already admitted nested
collection source without granting that source a child `default`. Standalone
alias codecs remain open; typed DTO/HTTP/SSE consumers retain their existing
checks. Source48, checked16, value/source16384, default32KiB, enum240 and JVM240
limits remain. No persisted-format change or default inheritance is introduced.

**Compatibility:** eligible formerly rejected source/factory inputs now
generate. Existing accepted call sites, wire behavior and ownership/lock formats
retain their contracts. This extends D01/D03/F04/F07/M03/M04/M07/M08; general
default graphs, platform matrices and final candidate acceptance remain open.

[Core contracts](../core/borrowed_enum_collection_defaults_test.go),
[bounded fuzz](../core/borrowed_enum_collection_defaults_fuzz_test.go),
[actual factory authority](../internal/schema/borrowed_enum_collection_defaults_test.go)
and [native preservation](../internal/cli/borrowed_enum_collection_defaults_process_test.go)
define the profile. The existing `string-enum-collection-bounds` feature consumer
also exercises independent original/alias factories, fresh state, optimization,
locks, four Json configurations and existing HTTP/SSE boundaries. Runtime and
platform evidence must match the exact generated consumer; prepare-only does
not prove compilation or execution.
