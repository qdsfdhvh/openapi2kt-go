# DTO enum-field uniqueness

The [ordinary DTO uniqueItems profile](dto-unique.md) also accepts direct
String enum fields and checked native Boolean/Int/Long/Float/Double enum
fields. The collection keeps its actual List/DTO API in both Data and Plain
styles. Validation is required; field references, nested DTO/container fields,
composed/hierarchy/projected/count/root-factory roles keep separate boundaries.
Direct scalar const bodies use the [const-body profile](dto-const-unique.md).

String enum getter keys use the actual enum's serialName. A known member and
Unrecognized("ready") compare equally when both carry "ready"; distinct unknown
strings remain distinct. Nullable fields and elements retain their original
views. Native enums retain their existing mapped membership and bounds checks;
floating keys normalize signed zero and reject non-finite values. This compares
the mapped DTO domain, including codecs' unknown-value behavior.

Own field defaults validate against the original vocabulary before required
initializer suppression. Omitted optional fields use the actual constructor's
initializer or null when computing collection-default keys. Thus items with
omitted state default"ready" and explicit state"ready" are duplicates, reported
at the second original default element Pointer. No target or child annotation
is inherited. JSON encodeDefaults/explicitNulls settings do not alter getter
keys, required/Patch presence or duplicate checks.

The analyzer validates the original field vocabulary without predicting an
emitted classifier. Final admission requires exactly one retained enum nested
under the actual DTO constructor, with the original enum node/Pointer, actual
property classifier and exact member names/wire vocabulary. Missing, repeated,
copied or changed declarations and same-byte source replacements cannot grant
permission. The runtime descriptor records an enum accessor role; original
source, constructor defaults and classifier provenance remain private.

Existing source48/checked16/JVMslots240, enum240, shared16,384-work/8MiB source
capture and16,384-work default-key limits remain. Repeated source captures and
canonical getter/type fingerprints keep their existing bounded reuse.

`bash scripts/test.sh feature dto-enum-unique` generates Data/Plain with the
keyword omitted for unchecked baselines and retained under validation. It
checks baseline and combined optimized/locked output, named/inline/nullable
DTOs, own defaults, known/Unrecognized equality, native membership, signed
zero, JSON configuration combinations, Patch, HTTP duplicate responses and
fragmented SSE failure/closure/client reuse. The wide profile retains240 DTO
fields and240 referring array fields with an actual enum getter. Go source
checks cover original identity, vocabulary correspondence and exact capture
budgets; native CLI checks cover core parity, readonly/no-op and failure
preservation. Broader graphs and platforms require separate evidence.

**Migration:** eligible enum-field DTO arrays formerly rejected by uniqueItems
now generate under validation. Existing accepted Kotlin APIs, wire behavior and
manifest/name-lock formats retain their contracts.

[Native enum collection getters](dto-native-collection-unique.md) separately admit original finite inline List/typed Map fields with existing numeric/Boolean enum leaves under validation. Actual native checks, vocabulary/default/initializer/source authority and mapped equality remain; other enum/ref/nominal leaf roles stay separate.

[Plain String enum collection getters](dto-string-collection-unique.md) separately admit original finite inline List/typed Map fields under validation. Abstract serialName descriptors require the final actual nested enum source/classifier/vocabulary/declaration proof; references, aliases, formats/bounds and other unadmitted leaves stay separate.

[DTO native enum reference getters](dto-native-reference-unique.md) separately admit already-supported numeric/Boolean enum scalar paths under validation, preserving actual native export/check/vocabulary and original chain/type/initializer authority. Existing nullable/default-role and no-inheritance contracts remain.

[DTO String enum reference getters](dto-string-reference-unique.md) separately admit already-supported plain named/internal/bridge enum sources under validation. Actual original enum and intermediate alias identity/physical types, strict own defaults and open serialName runtime behavior remain; other reference roles retain their contracts.
