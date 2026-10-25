# DTO scalar reference getter uniqueness

Under validation, ordinary DTO uniqueItems accepts fields whose original local
reference already maps to a supported builtin scalar: String, Boolean, Int,
Long, Float, Double, Uuid, LocalDate or Instant. Existing named/internal finite
chains and root bridges retain their type admission, nearest nullable intent,
actual primitive checks and original source roles. This permission concerns
DTO getter equality; it grants no new reference path or backend type.

Keys compare actual mapped getter values. Float/Double signed zeros normalize,
nonfinite values reject, UUID values and equivalent Instant offsets compare
equally. Required/optional and nullable fields retain their states. Named and
inline ordinary DTOs can mix these getters with existing direct scalar, enum,
const, JsonNull and inline collection getter profiles.

Field-own defaults reuse the same original grammar, value budgets and literal
rendering as deferred referenced-field defaults. Omitted fields contribute the
actual final initializer; an explicit equal value is a duplicate at the second
outer annotation value Pointer. Required annotations validate before initializer
suppression. Target/alias component factories never become consumer field
defaults. Existing target/hop default restrictions remain: an ordinary reference
without own default can consume a factory-bearing target, while a field-own
annotation must satisfy its existing independent source grammar. Patch presence,
Kotlin constructor/copy APIs and codec/wire behavior retain their contracts.

The private proof captures original reference-chain trees and physical paths
from the input root, including escaped internal keys. It retains schema nodes,
member/item identities, original catalog owners, alias binding/height/null intent,
actual mapped type/check data and initializer/null policy. Final verification
requires the retained scalar aliases and their recorded builtin/temporal
origins, intermediate exports, internal owners and original document paths.
A cached lookup, same-byte replacement node/ancestor/declaration, changed getter
checks or altered deferred initializer cannot authorize equality.

Capture work shares the existing16,384/8MiB document budget. Original nodes and
snapshots are memoized, while every traversed reference/path entry is charged;
retained original tree bytes are charged before storage. Reusing the same DTO
descriptor avoids a second capture. Source48, default-key16,384work, whole-value
16,384nodes/32KiB, checked16 and JVMslots240 remain. Reference cycles/missing
sources fail through the existing resolver. No source tree is expanded or
rewritten, and no type/enum/serializer is invented.

Enum/const/collection/DTO/raw/caller-native/value-class reference getters remain
separate, even when Kotlin spelling resembles a builtin scalar. Existing
approximation/open-enum contracts are unchanged. Output ownership, deterministic
shape identities and manifest/name-lock formats retain their current rules.

`bash scripts/test.sh feature dto-scalar-reference-unique` freshly generates
Data/Plain baseline/validation and combined optimized/locked output. Four Json
configurations exercise numeric/Boolean/UUID/time references, named/internal/
escaped bridge chains, own defaults, nullable overrides, required suppression,
component-factory non-inheritance, named/inline DTOs, actual mapped equality,
Patch, duplicate HTTP responses and fragmented SSE failure/closure/reuse.
An independent input compiles240 referenced scalar getters and240 referring
arrays; the existing independent depth16 collection probe remains. Source tests
replace target/ancestor/type/declaration/initializer states and check exact
capture work/bytes and descriptor reuse. Native CLI tests compare core output,
readonly/no-op and invalid annotation/target failures preserving old output.
Bounded fuzz checks deterministic options and omitted/explicit initializer keys.
Broader graph/platform/final candidate acceptance remains separate.

**Migration:** eligible DTO uniqueItems inputs with already-supported builtin
scalar reference getters formerly rejected now generate under validation.
Existing accepted APIs, wire behavior and persisted formats retain their rules.

[DTO native enum reference getters](dto-native-reference-unique.md) separately admit already-supported numeric/Boolean enum scalar paths under validation, preserving actual native export/check/vocabulary and original chain/type/initializer authority. Existing nullable/default-role and no-inheritance contracts remain.

[DTO String enum reference getters](dto-string-reference-unique.md) separately admit already-supported plain named/internal/bridge enum sources under validation. Actual original enum and intermediate alias identity/physical types, strict own defaults and open serialName runtime behavior remain; other reference roles retain their contracts.

[Referenced builtin collection getters](dto-builtin-collection-reference-unique.md) have a separate validation-only List/typed Map contract with original alias backing/declaration/type/check/initializer authority. This retains the scalar role and broader reference boundaries.
