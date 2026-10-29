# String enum metadata in models

String enum schemas may use the existing empty/uuid/date/date-time/byte/binary
formats and safe `x-native-type` metadata. In model contexts these remain lexical
String enums: the vocabulary `enum:["ready","paused"]` with `format:"date"`
encodes those words, keeps `serialName`, `fromSerialName` and open runtime
`Unrecognized`, and creates no date, byte-array or provider dependency.
Simple direct HTTP inline enum projections retain their separate scalar codec
and approximation warning; a reference to an actual model enum keeps its lexical
codec in HTTP and SSE. This extends D01/D03/D05/D06 and F04/F07/F08/M03/M07/M08/M12.

## Nullability and references

Named metadata enum roots may declare nullable intent. Pure named aliases and
already eligible finite named/internal reference paths retain the nearest explicit
intent; `nullable:false` narrows a nullable use without changing the original
physical enum class. JSON null remains distinct from the word `"null"`.
Metadata adds no null vocabulary member or nullable inference: those continue to
use the existing [plain String enum profile](string-enum-null-members.md).

Internal property, collection-leaf and composition-owned paths require the
original unique retained lexical declaration and its actual source/owner. Aliases,
List/typed Map trees, borrowing and tuple members reuse that classifier and codec.
Copied/discarded/ambiguous declarations do not acquire ownership. Existing graph,
recursive, composition and source admission rules still apply.

## Defaults and presence

The existing scalar, finite collection and object-default roles also admit
metadata String enum leaves. Direct fields, original component-own factories,
independent named component aliases and eligible reference-consuming fields keep
strict membership against the original String vocabulary, even though runtime
unknown strings stay open. Empty and nullable values still validate the complete
source vocabulary and metadata first. A default such as `"ready"` does not become
a date, UUID or provider literal because of metadata.

Each annotation is independent. A component factory, intermediate alias factory,
field initializer and outer object/collection annotation may coexist; borrowing
an independently annotated metadata enum or finite metadata enum collection does
not inherit that default. Unannotated optional getters retain their existing
presence. Required arguments keep no initializer, and ordinary nullable request
fields retain Patch.Absent/Null/Value. Invalid annotations reject before required,
Patch or discarded-field suppression, at the original value Pointer. Factories
and typed helpers decode fresh state using the final lexical serializer.

[Internal lexical enum alias factories](reference-wrapper-component-defaults.md#internal-string-enum-targets) also permit the alias's own annotation through existing retained internal declarations, with independent original field and consumer defaults.

[Direct String enum field bounds](string-enum-field-bounds.md) and existing direct collection leaf bounds also accept lexical metadata with validation. Other bounds and source roles remain separate.

Child/element-owned annotations, new default graph roles, value classes,
const/composition and unowned internal targets retain their separate eligibility.
Format/native metadata is validated even when an annotation is empty or null.

## Uniqueness and optimization

Under validation, existing String enum uniqueness roles compare actual
`serialName`, including eligible scalar references, finite List/typed Map trees
and supported ordinary DTO getter keys. Known values and same-name Unrecognized
values collide. UUID-looking words with different case remain distinct Strings;
metadata does not grant UUID, temporal, numeric or provider equality. Default
uniqueness uses the same original lexical keys and original duplicate Pointer.

Deduplication, hoisting, merging and name locks retain actual enum ownership and
codec/default identity. Equivalent component enums may become typealiases.
Companion lookup still works through the alias, and each component's named
`<ClassName> default` function remains available, including nullable factories.
Ordinary object factories retain their companion `default()` API.

**Compatibility:** formerly rejected eligible metadata inputs now generate.
Merging previously dropped a component enum's top-level factory when replacing
its class with an alias; generated aliases now retain that factory. Existing
accepted lexical codecs and manifest/name-lock formats remain. Regenerate owned
output and check drift when adopting the corrected optimized factory output.

## Validation and limits

Metadata/vocabulary parsing reuses the original shared builder. Unsupported
formats, malformed native spellings, invalid/duplicate/non-String vocabularies,
unsupported bounds or owner roles reject at original source locations. Original
schema depth48/work limits, enum240, component/container/object annotation
value16384/32KiB, checked16 and JVM240 limits
remain; tuple/borrowed captures retain their existing cumulative budgets. Source
identity and cached alias heights remain authoritative before optimization.

[Core model/default/uniqueness/merge contracts](../core/metadata_string_enums_test.go),
[factory/source guards](../internal/schema/metadata_enum_defaults_test.go),
[bounded fuzz](../core/metadata_string_enums_fuzz_test.go) and the
[synthetic fixture](../tests/fixtures/http/tuple-inline-enums.json) define this
profile. `bash scripts/test.sh feature tuple-inline-enums` exercises fresh
Data/Plain × validation, baseline and combined optimization/locks, four Json
settings, independent factories/defaults, nullable/internal codecs, 240-field
consumers, scalar/tuple Patch, HTTP and fragmented SSE failure/close/reuse.
These JVM checks do not establish additional target platforms or the complete
remaining feature and release matrix.
