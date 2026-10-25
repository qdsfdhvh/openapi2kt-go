# DTO temporal and UUID collection reference getter uniqueness

With validation, the existing [DTO collection reference role](dto-builtin-collection-reference-unique.md) also admits already-supported `Uuid`, `LocalDate` and `Instant` leaves. Sources use the existing `string` formats `uuid`, `date` and `date-time` within finite List/typed Map trees, through supported named aliases, internal collection sources and bridges. This follows the existing [mapped scalar equality](temporal-unique.md) contract; it adds no temporal bounds or arbitrary formatted-string admission.

The key compares actual typed getter values. UUID spelling variants map to the same UUID; equal Instant offsets/fractions map to the same instant. List order, Map key-order independence, nullable elements and the distinction between null and an empty collection follow the existing collection key. Data and Plain constructors, decoder checks, HTTP and SSE consumers share that behavior.

Field-own defaults retain the strict original [referenced temporal collection](referenced-temporal-collection-field-defaults.md) and UUID value grammar, original error Pointers and existing value/work/byte limits. Invalid annotations reject before required initializer suppression. Omitted defaults and explicit mapped-equal values contribute equal keys; JSON annotation spelling and typed key normalization stay separate. Target/alias defaults are not inherited or newly admitted by this role. Nearest nullable, existing direct dictionary views, required and Patch contracts remain.

## Source and type authority

The analyzer reuses the original scalar source/type vocabulary rather than admitting a formatted spelling alone. The early key descriptor is separate from the real forward reference. Final original alias declaration, Named backing address, actual getter/initializer, named/borrowed nested type/check snapshots, raw reference chains and physical source path/node/format identities remain authoritative. Changing a format marker, replacing it with a same-byte node, losing an origin or projecting the mapped type to text rejects. No source AST or catalog backing is fabricated or replaced.

All existing limits remain: source48 with complete incoming/cached hops, capture work16,384/bytes8MiB shared across fields/references, key/value work16,384, rendered default32KiB, checked traversal16 and member/JVM slots240. The existing strict value grammar, canonical aliases, serializers and ownership/name-lock versions stay intact.

## Boundaries and checks

Reference leaves inside the target tree, enums, const/null-only/nominal/raw/native/valueclass leaves, composed/projection roles, target/alias factories and new temporal bounds remain separate. General typed/nominal/projected uniqueness and broad reference/default/platform completion stay open.

[Core controls and mapped/default diagnostics](../core/dto_special_collection_reference_unique_test.go), [actual format/type/initializer source authority](../internal/schema/dto_special_collection_reference_unique_test.go), [bounded deterministic fuzz](../core/dto_special_collection_reference_unique_fuzz_test.go) and [native parity/strict-default failure preservation](../internal/cli/dto_special_collection_reference_unique_process_test.go) cover this extension. `bash scripts/test.sh feature dto-special-collection-reference-unique` freshly runs Data/Plain validation-off/on controls, optimized locks, 240 UUID collection reference getters with240 referring arrays, independent full checked-depth Instant references, four Json settings, mapped defaults, Patch, HTTP and fragmented SSE. The complete HTTP compilation retains this consumer; preparation alone is not execution evidence.

**Migration:** these eligible validated DTO uniqueness inputs formerly rejected the referenced collection field and now generate. Existing accepted Kotlin APIs, JSON/HTTP/SSE behavior, output ownership and persisted locks retain their contracts. This is a bounded D05/M03/M07/M12/V05 extension, not completion of the broader roadmap.
