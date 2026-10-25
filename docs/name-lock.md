# Hoisted-name lock: initial subset

[直接嵌套 native enum 集合字段默认值](nested-native-enum-collection-defaults.md)使用实际 scalar/container TypeRef、独立 nullable/check tree 与有效 canonical default 身份。每层 Map key 重排保持输出/锁；改变有效 default 或 retained checks 不能复用不同 frozen shape。Helper 绑定最终 dedup/merge/hoist owner/alias，不做 Kotlin 文本替换；required/Patch 无 initializer 的注解不新增 runtime default identity。strict source default membership 与 mapped runtime check 身份分开，原 occurrences、persisted format、ownership 和 CAS 不变。

[Referenced temporal collection field own defaults](referenced-temporal-collection-field-defaults.md) retain actual alias/leaf codec/nullability/check identity and original temporal lexemes alongside canonical Map keys. [Referenced String enum collection field own defaults](referenced-string-enum-collection-defaults.md) keep the same identity rules with original enum vocabulary membership; unknown supplied values stay open and no-default fingerprints are unchanged. [Referenced native enum collection field own defaults](referenced-native-enum-collection-defaults.md) apply the same rules to scalar-wire vocabularies with exact numeric default membership before mapping. Changed effective lexemes reject frozen shapes; required/Patch annotations without initializers add no runtime identity. No-default fingerprints and persisted state/CAS are unchanged.

[Referenced standard temporal scalar field own defaults](referenced-temporal-scalar-field-defaults.md) retain actual LocalDate/Instant source/type/nullable/codec identity and original default lexemes. Timezone-equivalent lexemes do not collapse; changed effective initializers reject frozen shapes. Required/Patch annotations without initializers add no runtime identity; historical no-default state and CAS remain unchanged.

[Referenced scalar field own defaults](referenced-scalar-field-defaults.md) retain actual builtin/check/nullability and effective literal identity. Mapped-equivalent literals can reuse a shape; changed initializers/checks reject frozen shapes. Required/Patch annotations without initializers add no runtime default identity. Existing opaque state/profile/allocation/occurrence and CAS rules remain.

[Direct nested temporal defaults](nested-temporal-collection-defaults.md) retain actual LocalDate/Instant types and original temporal strings in effective initializer identity. Canonical key reorder replays the lock; timezone-equivalent source lexemes are not collapsed. Required/Patch annotations without initializers add no runtime default identity, and opaque state/occurrence formats are unchanged.

[String enum collection bounds](string-enum-collection-bounds.md) retain checks on actual nominal leaf TypeRefs. Those field checks, nullable views and effective defaults participate in existing shape identity after lexical owner rebinding; changed bounds/pattern/defaults reject frozen shapes. Existing unbounded fingerprints and persisted formats remain unchanged.

[Referenced builtin collection field defaults](referenced-collection-field-defaults.md) keep actual alias/type/check identity alongside canonical effective initializers. Every dictionary level ignores key order; changed effective defaults/checks reject frozen shapes, while required/Patch annotations without initializers add no runtime identity. The version1 opaque state format is unchanged.

[Finite nested builtin defaults](nested-builtin-collection-defaults.md) use the final typed container/scalar/check tree and effective canonical initializer in the existing default-bearing shape domain. Each dictionary level ignores source key order; changing a real default or retained deep check still rejects a frozen shape. Required/Patch annotations without emitted initializers do not create defaults. No persisted format, CAS, ownership or lifecycle behavior changes.

[Direct nested String enum collection defaults](nested-string-enum-collection-defaults.md) use the actual lexical enum TypeRef/vocabulary, independent nullable/count tree and effective canonical default in that identity. Helpers bind the final owner after dedup/merge/hoist. Key reorder at every Map level replays the lock; changed effective values, non-null String vocabulary/order, nullable intent or retained counts cannot reuse a different frozen shape. Required/Patch annotations without initializers add no runtime default identity. Existing no-default fingerprints, persisted formats, occurrences and CAS remain unchanged.

[Interrupted transactions](recovery-rejection.md) cause read/generate/check/locked-validation rejection before naming-lock state is used, preserving old metadata and recovery evidence.

[Scalar DTO defaults](model-defaults.md) participate in shape identity through a separate fingerprint domain for previously rejected default-bearing declarations. Existing no-default fingerprints and the version1 opaque-hash state format remain unchanged; changing an emitted default triggers the existing shape-change rejection.

This interface stabilizes names for the [initial ordinary operation-root hoisting
subset](hoist-models.md), not arbitrary nested/union/hierarchy/error shapes.
`GenerationOptions.LockHoistedNames` is default false and requires
`HoistSharedShapes`. Pass `GenerationRequest.PreviousNameLock` bytes when reusing
state. Nil explicitly starts a fresh state; nonnil empty bytes are malformed.
Previous bytes with locking disabled return `E_OPTIONS`.

For explicit migration planning, `GenerationRequest.ProjectPreviousNameLock`
retains original allocations for a subset of original occurrences, including a
singleton canonical hoist. It requires previous locked state and cannot allocate
new locked shapes. The core grants no ownership: the adapter must separately
verify exact whole-source Kotlin and allocation-occurrence union coverage; see
[split adoption](ownership-migration.md#split-locked-owner-adoption). Ordinary
unprojected state still rejects removed occurrences; ordinary replay of an
already projected singleton preserves its canonical name. Existing encodings are
unchanged; older generators do not implement singleton retention.

`GenerationArtifacts.NameLock` contains canonical version1 JSON when enabled and
is nil otherwise. Both core generation APIs return the same state. Failed calls
return no files, state or warnings. The core does not load/save files, contact a
server or inspect environment variables. Callers must not mutate request bytes
concurrently with generation.

CLI/config persistence uses the explicit transactions below. Internal output
packages are not a public library API; ordinary internal output entry points still
reject nonnil lock metadata rather than silently discard it. Keep metadata and
its manifest with the generated Kotlin; do not append JSON as a Kotlin artifact,
write it after committing output or delete it to bypass a safety error. Broader
shapes remain unsupported; [explicit root moves](output-root-move.md) preserve
existing v1–v4 state, with selected-owner and disjoint source-group profiles.
Explicit [named-owner
rename](output-owner-rename.md) preserves version2/4 Kotlin and lock semantics,
and the disable/reset workflow below covers existing version3/4 output only.
This is not complete O03/T02.

## CLI and configuration

`generate` and `validate` accept `--lock-hoisted-names[=true|false]` (default
false; bare form is true). Locking requires effective `--hoist-shared-shapes`.
Values other than exact inline `true`/`false`, repetition and a separate boolean
argument are usage errors. File and explicit stdin input share these generation options; the downloader is removed.

```sh
openapi-kt generate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names
openapi-kt generate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names --check
openapi-kt validate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names
```

Direct locked validation requires `--output` as its read-only state root. Output
and suffix arguments on `validate` are available only with locking enabled.
Match the generation package/suffix/qualification profile; validation's neutral
package default is not a substitute for the stored profile. Validation reads
previous state and checks core compatibility, but does not plan writes or promise
a concurrent CAS snapshot. Fresh validation creates no directories or state.

Configuration accepts boolean `lock_hoisted_names` in defaults/specs. False
replaces inherited true; a CLI override requires one selected spec. The complete
effective inventory checks the hoist dependency before reading selected inputs
or state, including invalid unselected settings. Relative output paths use the
config directory. Locked specs load only their selected state's bytes; all
selected generation and warning policy complete before output lock acquisition.
The writer validates retained owned metadata and uses under-lock CAS.

Direct commands use manifest version3 when locked; configuration uses version4
for per-owner locks. Unlocked fresh/default roots retain version1/2. An existing
direct locked root cannot implicitly turn locking off; a selected configured
owner with state returns `E_NAME_LOCK_DISABLED` if locking is turned off. This is
not an implicit reset/migration flag. Use `--name-lock-action` below for explicit
disable/reset. Unselected locked peers remain retained; full
reconciliation may remove their owned metadata. Version4 is not silently
downgraded, even after its last locked owner is reconciled away. Version1
configuration adoption still requires explicit exact-match `--adopt-legacy`;
version3-to-version4 uses the separate [exact single-owner adoption](ownership-migration.md#locked-owner-adoption) workflow.

Storage failures are exit4; profile/shape/version compatibility failures from
core are exit1. Check differences are exit3 and no-op is exit0. The top-level
`files` list contains only Kotlin; explicit [owner rename](output-owner-rename.md)
adds a separate `owner_renames` metadata-preservation report. `--metrics` adds
entered `name_lock_read` timings; metadata is excluded from input/planned Kotlin
file-byte counts. Warning denial precedes
output mutation/acquisition, not the read-only state load. Generated sources,
metadata and manifest are never committed as separate transactions. Generation
and validation accept local file/stdin bytes only; no downloader or download environment bindings remain.

## Explicit disable/reset lifecycle

`generate` and `validate` accept the invocation-only
`--name-lock-action reset|disable` (also `--name-lock-action=reset`). Exact lowercase
values are required; repetition, empty values and unknown actions are usage errors.
There is no persistent action key in TOML. Keep effective hoisting enabled and
match the stored package, suffix and qualification profile for either action.
Version3-to-version4 adoption uses the separate [single-owner contract](ownership-migration.md#locked-owner-adoption). Complete same-version
output-root relocation uses the separate [root move](output-root-move.md) action;
the separate [named-owner rename](output-owner-rename.md) workflow is available only
with configuration and unchanged Kotlin/lock semantics. An action does not adopt
a fresh/unlocked tree or repair invalid state.

- **Reset** requires effective `lock_hoisted_names = true` / `--lock-hoisted-names`.
  It validates the old owned state, generates a fresh allocation from the current
  input without its reservations, and replaces code, metadata and manifest in one
  transaction. Shape drift or occurrence removal may now produce a reviewable
  candidate; malformed, missing, edited or unsupported old state still fails.
- **Disable** requires effective locking false. It regenerates the same fresh
  hoisted allocation, removes only the owned lock payload and changes direct
  version3 to version1 explicitly. Named output remains version4, even after its
  last lock is removed. Later ordinary generation allocates without old reservations.
  Unchanged Kotlin bytes retain their modification times.

```sh
# Review the reset candidate, then apply with the same input and options.
openapi-kt generate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names \
  --name-lock-action reset --check --diagnostic-format json
openapi-kt generate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names --name-lock-action reset

# Explicit disable; keep the stored hoist/qualification profile.
openapi-kt generate --input openapi.json --output generated \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names=false \
  --name-lock-action disable --check --diagnostic-format json
```

For configuration, set selected specs' `lock_hoisted_names` to the intended final
mode, then use `generate --config FILE [--spec NAME] --name-lock-action ACTION
--check`. A single-spec CLI boolean override is also available. The action applies
to every selected spec; a full multi-spec invocation must satisfy its target mode
for all selected specs before input/state IO. Partial selection retains unselected
owners and shared runtime membership; full generation retains the existing
reconciliation rule. An action cannot be combined with `--adopt-legacy`,
`--adopt-locked-owner`, `--move-output-from`, `--rename-owner` or the empty-inventory `--reconcile` flag, and every selected
owner needs existing state.
After disable, keep locking false in configuration for subsequent ordinary runs.

`--check` plans the entire candidate read-only and exits 3 for differences, 0 for
no-op. `validate` with an action generates the same report and performs read-only
output preflight, returning 0 for a safe supported candidate. Neither acquires a
writer lock, creates a root/stage, or modifies state. A plain locked `validate`
continues to check retained core compatibility without output write planning.

JSON adds `name_lock_migrations`, with one entry per selected owner (direct has no
`spec`): `action`, complete `before`/`after` hoisted inventories, sorted
`changes`, sorted `files`, and `impact`. Inventories record `role`, `shape`,
fully qualified public `classifier`, canonical `descriptor`, and original
`occurrences`. Descriptor names follow emitted top-level aliases after merging.
Changes correlate exact role+Pointer and list old/new classifier and descriptor,
plus `shape_changed`; absence denotes added/removed hoisted membership. File
changes compare the selected owner's old/candidate Kotlin membership and list
relative paths with old/new SHA256 (absence means membership addition/removal).
A path leaving one selected owner can remain on disk through a peer's shared
membership; unselected/reconciled owners are handled by the normal output plan.
Metadata remains outside the top-level `files` list. Text reports list
the same occurrence/file differences and migration guidance.

For example, adding an earlier operation while qualified locking retains
`models.AlphaResponseDto` can make reset allocate `models.AardvarkResponseDto`.
The old shared import is removed, API root aliases point to the new canonical
class, and serializer descriptors and nested classifier owners change. Migrate
shared-model imports and nested accesses, review constructor/validation changes
when shape hashes differ, and rebuild compiled consumers. The report is a
bounded allocation/file comparison, not an API or wire equivalence proof; moved
Pointers, split/merged groups and declarations leaving hoist eligibility require
review of the complete inventories and source diff. Disable can cause the same
immediate renames and also removes future name stability.

Lifecycle reporting uses the existing bounded fresh lock inventory even for
disable; it does not relax metadata or generation budgets. Old selected Kotlin
must be present, regular and hash-matching for a complete migration report.
All manifest-owned state, including retained/reconciled peers, is validated by
write planning. The exact old state bytes are compared before staging and again
under the complete sorted root lock set (`E_NAME_LOCK_STALE`); state is never
silently discarded or automatically retried. An action without existing owned
state returns `E_NAME_LOCK_LIFECYCLE` (exit4). Read-only reports are point-in-time
comparisons, not a concurrent snapshot or an approval token binding a later run.

Metadata removal/replacement shares the code/manifest backup journal, with each
manifest last. Every root stages before any apply; normal failures restore old
code, manifest and lock bytes/modes/mtimes. Obstructed rollback retains evidence
and returns `E_RECOVERY_REQUIRED`; an unfinished stage still refuses action,
check and validation. Handwritten files remain protected by normal ownership and
collision checks. Do not delete manifest/lock/stages to force a lifecycle action.

## Initial internal single-owner output transaction

The explicit adapter reads previous state without writing or acquiring a lock,
then requires the same expected bytes when committing the core artifacts. It
checks that precondition again under the cooperative writer lock before staging;
`E_NAME_LOCK_STALE` rejects stale generation instead of guessing or retrying.
Read-only checks create no output directory, lock or stage. They are point-in-time
comparisons, not snapshots against concurrent updates.

Only this entry point owns `.openapi-kt-names.json` through manifest **version3**:
exact fields `version`, `files`, `name_lock`, where `name_lock` is the metadata
file's SHA256. `files` remains exclusively safe relative Kotlin paths. Absent or
version1 output can enter this mode; an identical Kotlin file set needs only a
metadata/manifest update. Version2 named-owner output is not adopted. Ordinary
and named-owner writers reject version3 rather than silently downgrade it.

An existing unowned metadata destination is a collision, even for identical
bytes. Owned missing or edited metadata returns `E_NAME_LOCK_MISSING` or
`E_NAME_LOCK_MODIFIED`; malformed/unsupported state is not reset. Metadata reads
are limited to 1 MiB and reject symlinks/directories. Kotlin, metadata and the
version3 manifest use one staging/backup/journal transaction, with the manifest
last. Normal commit failures restore prior bytes, modes and file mtimes;
obstructed rollback retains backups and returns `E_RECOVERY_REQUIRED`.

The separate explicit lifecycle entry point can disable/reset through this same
transaction. This single-owner entry point has no implicit named-owner migration; explicit
[same-root adoption](ownership-migration.md#locked-owner-adoption) is a separate
metadata-only transaction. There is no automatic recovery or crash/power-loss
atomicity guarantee. Keep old metadata and
backups; do not delete them to bypass errors. Cooperative locking is not a
security boundary against hostile filesystem mutation. Existing version1/2
output without this explicit mode is unchanged.

## Initial internal named-owner / multi-root transaction

The explicit named adapter supports mixed locked and unlocked owners through
manifest **version4**: exactly `version`, `files`, `owners`, `name_locks`.
`owners` retains the version2 owner-to-sorted-Kotlin-path membership contract;
`name_locks` maps a subset of nonempty owners to metadata SHA256 hashes. Each
payload has the dedicated root-level filename
`.openapi-kt-names-<sha256(exact owner identifier)>.json`. Metadata is not in the
Kotlin `files` map. Owner identity is case-sensitive, as in version2; filename
hashing avoids freeform paths and reserved filenames.

Every selected locked owner supplies an expected-state map key, even for fresh
nil state. Explicit disable also supplies its old-state map key. Extra/missing
keys return `E_OWNER_UPDATE_INVALID`; absent and present
empty are distinct. The reader loads only the selected payload; the writer
validates all manifest-owned state before staging, including state being removed.
Corrupt, edited or missing state is not reset or deleted during reconciliation.
Individual payloads retain the core limits; a root independently permits at most
256 locked owners and 16 MiB of aggregate old or desired metadata. These limits
do not replace generation/work budgets or establish performance guarantees.

Unselected owners and their metadata remain unchanged unless explicit
reconciliation removes owners absent from the active inventory. Selected owners
with existing state cannot implicitly disable locking (`E_NAME_LOCK_DISABLED`),
even by generating an empty file set. Reconciliation removes only manifest-owned
metadata; similar unowned filenames are not scanned or deleted. Version4 remains
version4 even when its last lock is removed. A wholly unlocked invocation with no
prior version4 output retains the version2 format. Ordinary writers reject
version4 rather than downgrade it.

Version2 output can opt into this mode without rewriting unchanged Kotlin.
Version1 adoption requires the existing explicit, single-selected-owner,
exact-file-and-hash match; merely reading absent named state does not adopt
ownership. Version3 single-owner lock output requires separate [explicit exact adoption](ownership-migration.md#locked-owner-adoption). Explicit [named-owner
rename](output-owner-rename.md) transfers existing version2/4 membership in the
same root, preserving lock mode/profile/allocations/occurrences and exact Kotlin
bytes. It moves an owned payload to the new owner-derived filename using the
exact old bytes, compares manifest and every owned lock's bytes in preflight and
under all sorted root locks, and retains peers without reconciliation. Complete
same-version [root move](output-root-move.md) separately preserves v1–v4 raw
Kotlin/state/manifest bytes and attributes with full-owner inventory; partial
moves remain unsupported; version3-to-version4 uses separate exact adoption. Explicit selected
owner disable/reset uses the same version4 transaction with expected old state;
ordinary updates continue to refuse implicit disable.

For multiple roots, all sorted cooperative locks are held, every root's state
and retained peers are replanned, and all changed roots are staged before any
apply. A selected stale state returns `E_NAME_LOCK_STALE` without automatic
regeneration; newly committed unselected peers are retained. Metadata and Kotlin
share the existing backup/journal transaction with each manifest last and backups
retained until all roots succeed. Normal failures roll back applied roots;
obstructed restoration retains evidence and returns `E_RECOVERY_REQUIRED`.
Read/check are read-only point-in-time operations. This is not cross-root crash
atomicity, automatic recovery or hostile-filesystem protection.

Lifecycle verification adds [native action/report and source parity](../internal/cli/name_lock_lifecycle_process_test.go),
[configuration/retention/reconcile](../internal/cli/name_lock_lifecycle_config_test.go),
[actual merged descriptor reporting](../internal/cli/name_lock_lifecycle_merge_test.go),
[CAS/concurrency](../internal/cli/name_lock_lifecycle_unix_test.go),
[every commit/preparation boundary rollback](../internal/output/name_lock_lifecycle_fault_test.go),
[locked old-file validation](../internal/output/name_lock_lifecycle_old_files_test.go),
and [fresh baseline/reset Kotlin call-site, descriptor and wire contracts](../scripts/kotlin/name-lock-lifecycle.sh).
The standalone Kotlin command is `bash scripts/kotlin/name-lock-lifecycle.sh`;
`--prepare-only` runs native byte/report assertions and prepares projects without
executing a JVM check. It does not count as Kotlin compilation/runtime evidence.

## Version1 decisions

The exact envelope fields are `version`, `profile`, `entries`. Profile fields are
`package`, `suffix`, `qualified`. Each entry records `role`, `shape`, `name` and
sorted original schema-pointer `occurrences`. Entries sort by role+shape and
names are unique ignoring case. Only recorded eligible provenance authorizes
matching; allocated owner prefixes are not reverse-engineered.

`shape` is a lowercase SHA256 of the fixed `openapi-kt-owned-shape-v1` positional
protocol. It records model/enum kind, owned nested names, ordered properties and
vocabulary, wire/Kotlin names, types/default-null/checks/constant initializers,
nullability and property counts. Type/alias tags and check tuples are explicit,
not numeric Go enum values, Go type names, emitted Kotlin text or arbitrary IR
JSON layout. Only declared owned classifier names normalize to `$root`; opaque
provider/external types and codecs remain unchanged. Source work, diagnostic
pointers, presentation style and emission hints are not shape identity; every
original schema/constructor/validation budget runs before allocation.

The lock has independent metadata limits of 1 MiB, 4096 entries and 16384 total
occurrences. These do not change existing generation limits when locking is off.
Malformed/duplicate/unknown fields, invalid or unsorted entries, duplicate role
occurrences, unsafe classifier spellings and casefold duplicates are rejected.
Unsupported versions never reset silently. Changing the package/suffix/
qualification profile requires a deliberate migration, not implicit reuse.

## Reuse and failure behavior

All retained shared names reserve before allocating new groups. Reordering unique
operations, adding unrelated groups earlier lexically, or adding an earlier
canonical representative to an existing group leaves retained names unchanged.
New groups use existing component/runtime and retained-name reservations before
choosing numbered names. This preserves the shared class and descriptor name,
not arbitrary API method order or source-pointer rename recovery.

- `E_NAME_LOCK_INVALID`: malformed state; `E_NAME_LOCK_VERSION`: unsupported version.
- `E_NAME_LOCK_LIMIT`: metadata budget exceeded.
- `E_NAME_LOCK_PROFILE`: incompatible package/suffix/qualification.
- `E_NAME_LOCK_CHANGED`: an old occurrence changed shape, role or eligibility.
- `E_NAME_LOCK_REMOVED`: an old occurrence was removed.
- `E_NAME_LOCK_CONFLICT`: retained name conflicts with a component/helper or
  occurrence/shape matching is ambiguous.

The initial version refuses drift and removal instead of guessing a rename,
reclaiming a deleted name or splitting/merging historical entries. Starting with
nil is an explicitly new core state, not automatic corruption repair. The CLI
lifecycle action above validates old ownership and reports the fresh candidate
before transactional replacement. Preserve and review old source/lock state before
such a migration. Wire checks and the earlier hoist alias/nested-classifier
migration still apply. This is not general provider equivalence, crash atomicity,
an independent runtime library, platform coverage or a performance claim.

[Existing-v4 owner addition](ownership-migration.md#add-an-owner-to-an-existing-locked-inventory) uses `--adopt-locked-owner-from DONOR` with an explicit locked `--spec`. It projects donor allocations/occurrences and exact Kotlin subsets while retaining all old memberships/raw locks. Complete pre-input capture, all-peer CAS, existing budgets and manifest-last metadata-only rollback apply; existing full-inventory replay stays unchanged.
