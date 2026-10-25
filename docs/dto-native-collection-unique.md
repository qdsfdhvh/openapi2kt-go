# DTO native enum collection getter uniqueness

Under validation, the ordinary DTO uniqueItems profile accepts direct original
finite inline List/typed Map getters with existing numeric/Boolean enum leaves.
Int/Long/Float/Double/Boolean mappings retain their actual scalar codecs and
runtime checks. Plain String enum collection leaves, references/aliases,
const/nominal/raw/native-override/value-class/composed leaves and child defaults
retain separate boundaries.

Keys compare actual getter values. List order/length matter; Map key order does
not. Float/Double keys normalize signed zero and reject nonfinite values, with
existing mapped rounding. Native enum membership/bounds remain enforced by the
actual leaf check plan. Nullable leaves preserve their existing codec/source
rules, including the validation-only profile for explicit null vocabulary
members. The baseline without uniqueItems retains its original approximation.

Own collection defaults use the original field/default planner and strict
vocabulary before mapped-key normalization. Distinct permitted source literals
that map to the same Float compare equally. Omitted fields contribute their
actual constructor initializer; explicit equal List/Map trees are duplicates
at the second outer annotation value Pointer. Malformed vocabulary, bounds,
element defaults and invalid own defaults keep their original source Pointer.

DTO-key preflight calls the original native-enum field planner on isolated
checker/warning metadata. It borrows original schema nodes and adds no
provisional runtime checker or declaration. The actual retained getter type,
leaf checks, original vocabulary/source/Pointer and constructor initializer
must still satisfy the final DTO proof. Global scalar/structural uniqueness
rules and the other source-role permissions retain their contracts.

Shared checked16 includes current outer edges plus deepest getter field edges.
Source48, raw/member240 policies, serialized JVMslots240, source capture
16,384work/8MiB, default-key16,384work and whole-value budgets remain. Collection
nodes, values/map keys and omitted initializer work are charged; exact original
literal/descriptor reuse is cached. Ownership, deterministic fingerprints,
name locks, constructor/Patch and codec/wire APIs retain their existing rules.

`bash scripts/test.sh feature dto-native-collection-unique` freshly generates
Data/Plain baseline/validation and combined optimized/locked output. Four Json
configurations check all five native kinds, nullable values, Map-order and
floating-zero equality, own initializer reuse, invalid membership, nested
getters, named/inline DTOs, Patch, duplicate HTTP responses and fragmented SSE
failure/closure/client reuse. Independent probes compile240 native-enum List
getters with240 referring arrays, and15 getter edges plus one outer edge with
runtime JSON/duplicate assertions. Core/source tests check original errors,
Float rounding, actual retained checks/initializers, isolated probe metadata,
shared depth and exact default-key work. Native CLI tests preserve old output;
bounded fuzz checks deterministic options and omitted/explicit default equality.
Broader graph/platform/final candidate acceptance remains separate.

**Migration:** eligible DTO uniqueItems inputs with original numeric/Boolean
enum collection getters formerly rejected now generate under validation.
Existing accepted API/wire behavior and manifest/name-lock formats remain.

[Plain String enum collection getters](dto-string-collection-unique.md) separately admit original finite inline List/typed Map fields under validation. Abstract serialName descriptors require the final actual nested enum source/classifier/vocabulary/declaration proof; references, aliases, formats/bounds and other unadmitted leaves stay separate.

[Native enum collection reference getters](dto-native-collection-reference-unique.md) have a separate validation-only contract preserving the original vocabulary and actual alias/backing/check authority, strict own annotations and existing unchecked-mode gates.
