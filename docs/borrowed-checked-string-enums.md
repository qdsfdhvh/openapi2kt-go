# Borrowing checked lexical String enums

With `--validation`, existing internal schema references and finite collection
borrowing may use an actual lexical String enum that already owns String bounds
in an ordinary field or an admitted inline collection root. The original source
keeps `minLength`, `maxLength`, portable `pattern`, nullable intent and its actual
enum declaration. Supported format/native metadata remains lexical.

```json
{"Owner":{"properties":{"word":{"enum":["one","two","x"],
 "minLength":2,"maxLength":6,"default":"one"},
 "tree":{"type":"array","items":{"additionalProperties":{
   "enum":["one","two","x"],"minLength":2}}}}},
 "Word":{"$ref":"#/components/schemas/Owner/properties/word","default":"two"},
 "Tree":{"$ref":"#/components/schemas/Owner/properties/tree"},
 "Use":{"properties":{"word":{"$ref":"#/components/schemas/Word",
 "default":"one"},"tree":{"$ref":"#/components/schemas/Tree",
 "default":[{"a":"one"}]}}}}
```

Word aliases Owner's actual lexical enum. Tree retains the existing finite
List/Map backing and original enum owner. No new vocabulary, scalar projection,
provider dependency or component identity is fabricated for an internal node.
Forward references and finite named/internal chains use the same source and
final declaration proof as already admitted references.

## Checks, defaults and wire behavior

DTO constructors, Data copy and automatic decoding retain checks on the actual
enum `serialName`, including in typed List/Map/Patch consumers. Known `X` stays
an enum entry but fails a checked use's minLength. Runtime
`Unrecognized("future")` passes matching checks. Independent enum lookup,
serialization and transparent alias assignment remain open and are not intrinsic
validation boundaries. Existing serialName and typed-tree uniqueness apply to
eligible consumers; valid unknown values can equal known entries by serialName.

Original field, alias factory and consuming annotations validate independently.
Every non-null default leaf must belong to the original vocabulary and satisfy
its bounds. A valid runtime unknown word is still an invalid default. Field
annotations and opt-in factories do not become defaults on other owners.
Required/Patch annotations validate before initializer suppression. Each source
retains its original nullable grammar, while the consuming plan uses the nearest
explicit alias/field override. Null layers skip only their own checks.

An eligible internal lexical enum alias may retain its own factory and the
original ordinary field's own annotation. Object defaults and existing finite
collection own-default roles reuse the original strict vocabulary/checks with
the consuming `/default/...` Pointer. Map keys remain opaque data and sort after
source-order validation. Empty/null annotations still validate the original
source. Malformed checked-source errors retain the original keyword Pointer.
Borrowed collection source/factory annotations additionally follow the
[independent collection default contract](borrowed-enum-collection-defaults.md).
Child-owned annotations and other unadmitted factory/graph roles remain separate.

Typed HTTP scalar and collection bodies/responses preserve preflight/postflight
checks. Known SSE data uses the same retained checks; invalid fragmented input
fails and closes that input, while the caller's client and a fresh cold stream
remain reusable. Standalone alias codecs, caller mutation and request-builder
replacement retain their existing limits.

## Original source and enforcing field

A forward borrower and its original field share the same check object. Final
validation proves the original source node/Pointer/bytes, ordered vocabulary,
unique actual enum declaration and checked-source marker. Original enforcing
getters also retain their actual type/check addresses and check values. Keeping
a valid cache entry while dropping or replacing the source field's check does
not authorize output. Direct collection roots retain their existing actual
alias/backing/peer proof.

Copied, missing, repeated, discarded or ambiguous declarations retain their
existing rejection policy. Source ownership and actual nullable/check/default
identities survive optimization and naming locks; persisted formats remain
unchanged. The profile adds no general checked enum-root, const/value-class,
tuple, projection or intersection admission. Standalone bounded components use
the separate [checked lexical enum-root contract](checked-string-enum-roots.md).
Borrowing still requires an actual retained source declaration and check role.

Limits remain source48, checked collections16, source/value16384,
default32KiB, enum240, JVM constructor240 and input8MiB. Validation-off rejects
the constrained source. No heap, pattern dialect or runtime timeout changes.
Independent resource probes do not prove maximum width/depth together, all
platforms or minimum dependencies.

**Compatibility:** eligible formerly rejected borrowing/default inputs now
generate with their original lexical classifiers. Existing accepted APIs/wire
and persisted formats retain their contracts. Regenerate and compile new
consumers; invalid checked values throw at the existing typed boundaries. This
extends D01/D03/D05/F04/F07/M03/M04/M07/M08/M12 and typed HTTP/SSE consumption,
leaving general graphs and final acceptance open.

[Core borrowing/default cases](../core/borrowed_checked_string_enum_test.go),
[source/presence guards](../core/borrowed_checked_string_enum_guards_test.go),
[bounded fuzz](../core/borrowed_checked_string_enum_fuzz_test.go),
[original check/declaration/getter proof](../internal/schema/borrowed_checked_string_enums_test.go)
and [native failure preservation](../internal/cli/borrowed_checked_string_enum_process_test.go)
define this profile. The existing `string-enum-collection-bounds` fixture and
consumer cover fresh Data/Plain checked and unbounded-off controls, four Json
settings, actual factories, Patch, optimization/locks, independent resource
probes and direct/shared HTTP/fragmented SSE. Prepare-only is generation/check
evidence; Kotlin evidence requires compilation and execution.

The [five-input platform probe](platform-enum-field-bounds.md) also includes
checked scalar and Map/Map borrowing, independent alias factories and field
defaults, fresh state, open standalone aliases and enforcing original/borrowed
DTO consumers. Its common assertions use the same four Json configurations;
platform and dependency evidence remains scoped to that probe.
