# DTO inline scalar collection getter uniqueness

The [ordinary DTO uniqueItems profile](dto-unique.md) accepts direct inline,
finite List/typed Map fields with builtin scalar leaves. These fields can mix
with the existing scalar/enum/const/JsonNull getter profiles on the same actual
ordinary constructor. Validation is required. References, collection aliases,
enum/DTO/raw/native/value-class leaves and composed field paths retain
separate boundaries; the standalone structural collection profile is broader
and does not grant those roles to DTO getters.

Keys preserve the actual typed getter tree. List order matters; Map key order
does not. Nullable containers/elements retain their original states. Builtin
Boolean/Int/Long/String/LocalDate, Float/Double, Uuid and Instant leaves use
existing mapped values; floating zeros normalize, non-finite values reject,
and UUID/Instant compare parsed values. No reflection or Json re-encoding is
used. Existing field checks/constructors/default/copy/codec/Patch APIs remain.
Unknown source DTO keys already discarded by codecs are not recovered.

Own collection field defaults validate through the existing field planner.
Default keys reuse structural mapped normalization; omitted optional fields
use the actual constructor initializer. For numbers default[0], items with
omitted numbers and explicit[-0.0] are duplicates under Float mapping, reported
at the second original default value Pointer. A typed Map with reversed keys
and equivalent values likewise compares equally. Default data keys such as
"$ref" remain data. Required initializer suppression, child/default-role
rejections, original invalid-leaf/default Pointers and no inheritance remain.

Source admission follows only original inline schema edges, without guessed
declarations. Final proof requires the actual getter tree, original node/
member/Pointer identities and one retained DTO. Mapped leaf types, actual
constructor initializer expressions/null-initializer policy and canonical
initializer keys/work must agree with the validated original field plan.
Initializer plans stay private; the final barrier performs no new graph
admission. Raw projections, changed getters/initializers, same-byte item
replacements and unproved leaves cannot authorize equality.

The checked16 budget now includes both the current outer collection path and
the deepest DTO getter collection path. A List<DTO> key with15 field edges
reaches16;16 field edges reaches17 and is rejected at the original collection
Pointer. Named DTO boundaries do not reset this key traversal. Other nominal
constructor validation keeps its existing barrier. Source48/enum240/JVM
serialized-slots240 and source capture16,384work/8MiB remain.

Default keys share16,384 work per document. The DTO node and scalar/null slots
retain their existing charges; collection fields additionally count container
nodes, values and Map keys. Omitted collection initializers charge their cached
validated tree work for each distinct DTO key. Reusing an identical original
literal/descriptor key does not charge again; a different same-byte literal
does. Key normalization encodes each field tree once, and cached getter/type
SHA references keep shared trees out of repeated model fingerprints. Source,
initializer plans, canonical keys and work metadata remain outside Model identity.

`bash scripts/test.sh feature dto-collection-unique` runs Data/Plain baselines
without uniqueItems/validation-only bounds, plus validation and combined
optimized/locked output. It checks numbers/Map/nested List-Map getters,
initializer/Map-order/Float/UUID/Instant equality, JSON configurations,
nullable/Patch, HTTP duplicate responses and fragmented SSE closure/reuse.
Independent probes compile240 List constructor fields with240 referring array
fields, and15 getter edges plus one outer edge with runtime JSON/duplicate
assertions. These are independent widths/depths, not a Cartesian maximum.
Source tests cover actual tree/initializer identity, exact cumulative key work,
original failures and bounded canonical encoding for240×240 shared field trees
and a32KiB initializer referenced240 times. Native failure preservation and
bounded deterministic fuzz are separate checks. No broader platform/graph or
exact JVM feasibility claim is implied.

**Migration:** eligible DTO uniqueItems inputs with inline builtin scalar
collection fields formerly rejected now generate under validation. Existing
accepted APIs, wire behavior and manifest/name-lock versions remain. New getter
paths exceeding the shared checked16 budget reject without changing output.

The separate [JsonNull collection getter extension](dto-null-collection-unique.md) admits original direct plain/null-only enum leaves while retaining DTO-local source/type authority and existing nullable codec behavior.

[Native enum collection getters](dto-native-collection-unique.md) separately admit original finite inline List/typed Map fields with existing numeric/Boolean enum leaves under validation. Actual native checks, vocabulary/default/initializer/source authority and mapped equality remain; other enum/ref/nominal leaf roles stay separate.

[Plain String enum collection getters](dto-string-collection-unique.md) separately admit original finite inline List/typed Map fields under validation. Abstract serialName descriptors require the final actual nested enum source/classifier/vocabulary/declaration proof; references, aliases, formats/bounds and other unadmitted leaves stay separate.

[DTO scalar reference getter uniqueness](dto-scalar-reference-unique.md) separately admits already-supported builtin/temporal scalar reference fields under validation, with original chain/owner/actual declaration/type/initializer authority. Field-own defaults reuse the existing deferred default planner and source-role restrictions; consumers inherit no component defaults.
