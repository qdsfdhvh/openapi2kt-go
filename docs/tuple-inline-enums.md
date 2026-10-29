# Inline enum positional-array sources

OpenAPI 3.1 `prefixItems` members and schema-valued `items` tails accept direct
String vocabularies (including the metadata profile below) and native Int/Long/Float/Double/Boolean enums. The
shared collection planner supplies the selected element's actual type and codec.
This extends D03/D06 and F08/M03/M07/M08/Q05; positional arrays still use an
explicit List approximation.

```json
{"Rows":{"type":"array","prefixItems":[{"enum":["ready","paused"]}],
         "default":["ready"]},
 "Numbers":{"type":"array","prefixItems":[{"type":"integer","enum":[1,2]}]},
 "Raw":{"type":"array","prefixItems":[{"enum":["ready"]},{"enum":["ready"]}],
        "items":false,"default":[null,{"enum":"data"}]}}
```

A single String-enum prefix selects its original vocabulary. Two distinct inline
enum nodes remain different sources even when their values are equal, so the
last example uses `List<JsonElement>`. An explicit admitted schema tail wins
after every original prefix validates. `items:true` selects raw JSON;
`items:false` retains the prefix's uniform/mixed decision. Enum references keep
[their existing source and validation gates](tuple-native-enum-references.md).

In model collections, String enums retain their actual lexical sealed enum,
`serialName`, known values and open runtime `Unrecognized` strings. Explicit and
typeless plain String vocabularies retain existing nullable/null-member rules;
JSON null differs from the string `"null"`. Simple inline HTTP String-enum
collections retain their existing scalar projection and warning: plain Strings
remain Strings, while format/native metadata selects its actual scalar codec. Named
model references retain the actual model codec in HTTP and SSE.

Native enums retain their scalar wire type and numeric mapping. Validation-on
typed consumers retain membership checks and finite floating values; unchecked
consumers retain the existing membership approximation and warning. Native null
vocabulary members require the existing validation-on nullable source profile.
Transparent List aliases are not new constructor/check barriers. A raw projection
retains no positional membership check. `W_TUPLE_APPROXIMATION` describes the
lost positional/length constraints and possible narrowing of unrestricted tails.

## String format and native metadata

String enum tuple sources accept `format` values `""`, `uuid`, `date`,
`date-time`, `byte` and `binary`, and safe existing `x-native-type` spellings.
In model collections these remain lexical enums: `enum:["ready"]` with
`format:"date"` still encodes `"ready"`, accepts unknown runtime strings through
`Unrecognized`, and does not parse its vocabulary as dates, bytes or provider
values. Provider metadata alone creates no provider dependency for this lexical
model enum. Malformed metadata, unsupported formats, bounds, value-class flags,
position-owned defaults and invalid vocabularies reject at their original source
Pointer, including discarded prefixes.

Named metadata enum components and finite pure named aliases may also be used as
members or tails. Each use retains the actual canonical component, lexical
StringEnum declaration, original vocabulary, cached alias height and consuming
TypeRef address; a missing, copied, duplicated or mutated declaration cannot
authorize even a discarded prefix. A consumer may independently request a
nullable element. Metadata enum component nullability, aliases and eligible internal sources follow
the [model metadata contract](metadata-string-enums.md). Null-vocabulary
inference and general source/ownership rules retain their separate permissions.

Simple direct inline HTTP tuples preserve the operation planner's scalar codec:
for example, `date` yields LocalDate, `byte` yields signed ByteArray JSON arrays,
and native bigint yields numeric Long elements. Their enum membership remains
an explicit approximation. Named model enum references retain their lexical
codec in HTTP. Ordinary model/request object fields use their lexical nested
enums, including Patch and shared HTTP/SSE payloads.

Existing outer field/root/independent tuple alias defaults use strict original
String membership and the actual selected lexical codec; a non-date vocabulary
word remains a valid model enum default. Required and Patch annotations validate
before suppression. Target enums and named aliases may own independently validated factories under
the model metadata contract; a tuple annotation never inherits them. Positional
member defaults and new graph roles retain separate eligibility.

## Own annotations

Existing tuple outer field, original component root and independent component
alias defaults also accept selected direct enum elements. String annotations
must belong to the original vocabulary despite open runtime strings. Native
annotations compare the original numeric vocabulary before Float rounding.
Nullable elements and containers retain independent null intent. Required
arguments receive no initializer, Patch stays Absent, and no annotation is
inherited from a target, child or positional member. Each factory/helper decodes
fresh values with the actual retained serializer.

Raw/mixed enum prefixes use the [raw tuple default contract](tuple-raw-defaults.md):
annotations are raw JSON data, including null and metadata-looking object keys.
Source vocabulary validation still precedes empty/null annotations or projection.
Selected admitted reference elements use the [reference default contract](tuple-reference-defaults.md); other DTO/composed/default graph roles remain independent.

## Sources, limits and compatibility

Original schema/vocabulary/member nodes and physical ancestor paths are captured
and rechecked, including discarded prefixes. A same-byte copied schema or
literal supplies no original source permission. All newly admitted inline enum
sources share the existing enum-source capture limit: 16,384 node/edge/catalog
work and 8 MiB terminal bytes, with repeated original captures reused. Raw
vocabularies are limited to 240 members in either validation mode. Existing
source depth48, reference work16384, default value16384/32KiB, checked16 and
JVM240 limits remain.

Position-owned defaults, bounds, const, value-class
String enums, composed/container members and more
general tuple graphs retain their separate admission rules. Every original
unsupported key or malformed vocabulary rejects at its source Pointer, even
when a tail would discard it.

Previously rejected eligible inputs now generate. Existing accepted Kotlin
APIs, wire behavior and manifest/name-lock formats retain their contracts.
Enabling merge/hoist separately follows the existing classifier migration:
typealiases preserve companion APIs such as `fromSerialName`, while nested enum
constants require their actual canonical classifier.

[Core](../core/tuple_inline_enums_test.go),
[original source guards](../internal/schema/tuple_inline_enums_test.go),
[native lifecycle](../internal/cli/tuple_inline_enums_process_test.go) and the
[synthetic fixture](../tests/fixtures/http/tuple-inline-enums.json) define this
profile. `bash scripts/test.sh feature tuple-inline-enums` builds fresh Data/Plain
× validation off/on projects, baseline and combined optimized output with actual
nonempty hoist locks. The [consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleInlineEnumsContracts.kt)
checks four Json configurations, enum/null/native mappings, independent defaults,
required/Patch, typed and projected HTTP, live fragmented SSE and failure cleanup.
Independent 240-field native enum and named metadata enum tuple DTOs compile in
each profile. Metadata cases additionally exercise lexical versus temporal, byte
and native HTTP codecs, independent defaults and original declaration authority. The
complete `http` route includes the consumer. These checks do not establish the
full tuple graph, option Cartesian product or additional target platforms.

[Boolean positional literals](tuple-boolean-members.md) have their own typed false/raw source and default contract.

[Inline ordinary DTO tuple sources](tuple-inline-dto.md) have a separate actual-constructor and isolated source-validation contract.
