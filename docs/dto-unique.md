# Mapped scalar-field DTO uniqueness

With validation enabled, uniqueItems can check arrays whose elements are actual
ordinary DTO constructors with direct builtin scalar fields. Named components
and original inline constructors under direct lexical owners retain their
existing Kotlin List/DTO API. Nullable elements keep their original views and
at most one null is accepted. The homogeneous tuple profile can use this role
when its explicit tail provides the same already accepted DTO source; bare
prefix-only DTO tuples retain their separate uniqueItems boundary.

Keys contain actual mapped getter values, independently of data/plain class
style and Json encodeDefaults/explicitNulls. String, Boolean, Int/Long,
Float/Double, Uuid, LocalDate and Instant fields use their existing codecs.
UUID and Instant compare parsed values; floating fields compare mapped finite
values with signed zero normalized. A non-finite floating getter is rejected.
This compares the mapped DTO domain; it does not recover unknown source JSON
keys discarded by a configured codec.

Default annotations use the same mapped equality after ordinary object/leaf
validation. Omitted optional properties receive their actual constructor
initializer or null key. For Item(id required, version default0),
`[{"id":"a"},{"id":"a","version":0}]` is rejected at the second original
value Pointer. Required fields still require supplied values, while their own
annotations validate before initializer suppression. Duplicate defaults,
invalid primitive annotations and original bounds keep their source Pointers.
No target/intermediate/component default is inherited.

Source syntax grants only provisional admission. Original catalog/Pointer,
semantic node/member identities and one actual retained constructor must match
the final typed use and exact getter/type/default plan. A same-byte replacement,
missing/repeated/copied constructor, projected spelling or synthetic source
cannot authorize a key. Inline short names require the exact original items
scope and actual class name. Annotation-only object validation separately
requires the same actual constructor and cannot grant a typed JSON projection.

Direct JsonNull fields and null const bodies use the [JsonNull getter profile](dto-null-unique.md).
Direct enum fields use the [DTO enum-field profile](dto-enum-unique.md).
Direct builtin scalar const bodies use the [const-body profile](dto-const-unique.md).
Direct inline scalar List/Map fields use the [collection getter profile](dto-collection-unique.md).
Bytes/caller-native/valueclass fields, nested DTO/recursive field
graphs, field-reference expansions, composed/hierarchy owners and targets,
projected DTOs, target property counts/root factories and unproved alias views
retain separate boundaries. Already accepted scalar/enum/JSON/structural
uniqueness profiles keep their existing rules. No general nominal/projected
uniqueness or exact JSON Schema validation is claimed.

Original semantic source capture has a separate shared16,384 work/8MiB budget
per document. Repeated sources reuse capture; snapshots/model checks run once
per distinct source while each final element view is checked. Default keys
share another16,384 work budget with identical literal/descriptor reuse.
Original source48/checked16/JVMslots240 and default/value/input limits remain.
Constructor-default keys, wire/source nodes and provenance stay private to the
analyzer. Enforcing descriptors contain getter/type information; canonical
model fingerprints use a cached SHA256 reference so shared descriptor graphs
are not expanded for every collection occurrence. Private typed runtime checks
share only identical actual parameter types and check bodies.

`bash scripts/test.sh feature dto-unique` generates Data/Plain×validation
profiles, with the keyword omitted in unchecked inputs. It covers baseline and
combined optimized/locked output, named and inline DTOs, nullable elements,
constructor/default/Json checks, required/Patch, HTTP duplicate responses and
fragmented SSE failure/closure/reuse. An independent wide profile compiles
240-field DTOs and240 referring collection fields. Source tests check original
identity, actual constructors/defaults, exact capture/key budgets and bounded
canonical encoding for shared240×240 graphs and a32KiB own initializer referred
from240 fields. Native checks compare core bytes, readonly/no-op, original
error Pointers and failure preservation; bounded fuzz varies numeric formats,
nullable/style/options and omitted/explicit initializer duplicates. Broader
codecs/graphs/platforms require separate evidence.

**Migration:** formerly rejected eligible DTO uniqueItems inputs now generate
with validation checks and actual constructor/default/codec behavior. Existing
accepted APIs/wire behavior and manifest/name-lock versions remain. The new
runtime descriptor affects only this newly admitted enforcing role; source
provenance and initializer-only data do not enter semantic declaration identity.

[DTO JsonNull collection getter uniqueness](dto-null-collection-unique.md) separately admits finite inline List/typed Map getters with original direct plain/null-only enum leaves. It retains actual getter/initializer/source authority and existing nullable codec collapse; global structural uniqueness remains separate.

[Native enum collection getters](dto-native-collection-unique.md) separately admit original finite inline List/typed Map fields with existing numeric/Boolean enum leaves under validation. Actual native checks, vocabulary/default/initializer/source authority and mapped equality remain; other enum/ref/nominal leaf roles stay separate.

[Plain String enum collection getters](dto-string-collection-unique.md) separately admit original finite inline List/typed Map fields under validation. Abstract serialName descriptors require the final actual nested enum source/classifier/vocabulary/declaration proof; references, aliases, formats/bounds and other unadmitted leaves stay separate.

[DTO scalar reference getter uniqueness](dto-scalar-reference-unique.md) separately admits already-supported builtin/temporal scalar reference fields under validation, with original chain/owner/actual declaration/type/initializer authority. Field-own defaults reuse the existing deferred default planner and source-role restrictions; consumers inherit no component defaults.

[DTO native enum reference getters](dto-native-reference-unique.md) separately admit already-supported numeric/Boolean enum scalar paths under validation, preserving actual native export/check/vocabulary and original chain/type/initializer authority. Existing nullable/default-role and no-inheritance contracts remain.

[DTO String enum reference getters](dto-string-reference-unique.md) separately admit already-supported plain named/internal/bridge enum sources under validation. Actual original enum and intermediate alias identity/physical types, strict own defaults and open serialName runtime behavior remain; other reference roles retain their contracts.
