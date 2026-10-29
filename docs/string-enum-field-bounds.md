# String enum field bounds

With `--validation`, direct String enum fields in ordinary DTOs accept
`minLength`, `maxLength` and the existing portable `pattern`. Explicit String,
typeless and already supported nullable vocabularies retain their lexical enum
declaration. Supported format/native metadata also remains lexical: a
`format:"date"` enum containing `"one"` still encodes that word.

```json
{"properties":{"state":{"type":"string","format":"date",
 "enum":["one","two","x"],"minLength":2,"maxLength":6,
 "pattern":"^[a-z]+$","default":"one"}}}
```

The generated getter keeps its actual enum type. DTO construction, Data-class
copy and automatic JSON decoding check `serialName`; this applies equally to
known members and `Unrecognized`. `X` remains available in the enum, but supplying
it to this checked field fails minLength. A valid unknown word such as `future`
passes. Standalone enum lookup, serialization and decoding keep their open
runtime vocabulary and do not impose the field's constraints.

String length counts Unicode codepoints. Patterns use the existing portable
Regex substring-search profile; use anchors for whole-string matching. The
1024-byte/256-node/1000-quantifier limits and printable-ASCII compile-time pattern
matching policy remain. Invalid bounds and patterns report their original
keyword Pointer even when the field's default is null.

Own field defaults must satisfy both original vocabulary membership and retained
String checks. Runtime openness never admits an unknown annotation. Required
fields validate their annotations before suppressing the initializer; nullable
request fields retain Patch.Absent/Null/Value. Nullable null values bypass only
their field checks. Missing/null/default wire behavior and the existing Json
encodeDefaults/explicitNulls policies remain. No field annotation is inherited
from another owner.

The same lexical metadata is accepted on the existing [direct List/typed Map
String enum leaf bounds](string-enum-collection-bounds.md) role, with or without
an outer default. Metadata creates no UUID/date/bytes/provider equality or new
runtime dependency. Actual vocabulary/source nodes, validation Pointers, final
canonical enum owners and check/default identities remain through optimization
and naming locks. Existing counted ordinary DTOs can contain these fields;
whole-object count and scalar checks retain their independent boundaries.

This is a direct ordinary field role, including ordinary allOf constructors that
retain a single original field. Enum roots, referenced/borrowed checked enum
leaves, hierarchy/union fields, cross-contribution
aggregation and discarded/projection roles retain their
separate eligibility. Numeric constraints, const and value-class sources remain
excluded. Checks require validation; disabling it rejects the original bounds
instead of dropping them. Original schema-depth48, enum240, checked-edge16,
JVM240, default16384-node/32KiB and input8MiB limits remain.

**Compatibility:** eligible bounded fields and metadata collection leaves that
previously rejected now generate. Existing accepted APIs, lexical wire codecs
and manifest/name-lock formats retain their contracts. Regenerate new checked
inputs and compile consumers; invalid field values throw at the existing DTO
boundaries. This extends D03/D05/M03/M04/M07/M08/M12/V05 and typed HTTP/SSE
consumption, without establishing broader platform or final candidate acceptance.

[Core admission/default/Patch guards](../core/string_enum_field_bounds_test.go),
[bounded fuzz](../core/string_enum_field_bounds_fuzz_test.go),
[original source/check tests](../internal/schema/string_enum_collection_bounds_test.go),
[native failure preservation](../internal/cli/string_enum_collection_bounds_process_test.go)
and the existing collection-bounds fixture/consumer define the profile.
`bash scripts/test.sh feature string-enum-collection-bounds` runs fresh
Data/Plain with validation on and unbounded off controls, four Json settings,
known/unknown/null/default/Patch behavior, copy, optimization/locks, independent
240 collection and scalar-field probes, typed HTTP and fragmented SSE failure,
close and client reuse. Prepare-only execution proves generation/check only.


The [five-input platform probe](platform-enum-field-bounds.md) separately checks
this lexical field/metadata collection subset in Data/Plain JVM, JS/Node and
macOS ARM64 Native execution plus common metadata/Android AAR compilation.
It retains the four previous inputs and four Json configurations; device,
complete feature and minimum-version acceptance remain separate.


[Bounded enums in object defaults](bounded-enum-object-defaults.md) additionally
permit existing ordinary object walks to validate direct enum fields and direct
List/typed Map enum leaves. Original plain DTO component/alias factories and
consuming annotations remain independent; strict values, original Pointers,
source/check authority and existing budgets remain.

[Finite nested List/Map bounds](nested-string-enum-bounds.md) separately extend ordinary inline collection fields, outer defaults and existing object-default walks through the exact original lexical terminal.

[Checked lexical enum borrowing](borrowed-checked-string-enums.md) separately admits already retained ordinary field/collection-root sources through existing references, with the original enforcing checks and independent eligible defaults. Standalone checked enum components and other source roles keep their own eligibility.
