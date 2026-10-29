# Merge into an owned output root

The invocation-only `--move-output-from SOURCE --output TARGET` action also
accepts an already owned target for **same-version v2/v4 named inventories**.
Use `--config FILE` with the complete original source owner group and unchanged
inputs/options. Omit both selectors to move that entire group, select one existing
source owner with `--spec NAME`, or select a set for one atomic normal-failure transaction using repeatable `--move-owner NAME`. `--move-owner` requires config and the explicit SOURCE/TARGET pair, excludes `--spec`, and keeps the ordinary emission-override restriction. The ordinary [root-move contract](output-root-move.md)
continues to define source generation, selection, canonical bindings, cooperative
locks, normal rollback, reports and exit codes.

## Accepted inventories

Moved owner names must be disjoint from existing target owner names. All moved
Kotlin paths may be new or match an exact target-owned relative path with identical
original bytes. The latter joins owner memberships without rewriting the target
file: its original mode and mtime remain, even when source attributes differ.
Different bytes, case-only spellings and file/parent conflicts refuse. Identical
unowned destinations still collide; only verified target ownership permits sharing.
Existing source peers may continue sharing a moved file: the file remains at
source while a peer still owns it and is copied unchanged to target.

The source and target must both be v2 or both be v4. A v4 inventory may contain
locked and unlocked owners, including an empty lock inventory. Existing limits
apply to the combined target: at most 256 lock payloads, 16 MiB aggregate raw
locks, the existing per-lock limits, and the existing manifest byte/parser
limits. A valid root on each side does not imply their union fits those limits.
Direct v1/v3 roots, version conversion, duplicate moved owner identities,
owner reassignment/removal and incomplete source inventories
remain outside this profile. Refusals preserve both roots.

All existing target ownership is read and storage-validated before reading any
source input: raw manifest, every owned Kotlin file, all owned raw locks, modes
and mtimes. The target is not regenerated. Config may include original target
owners declared at exactly TARGET; those inputs are not read for this action.
A missing target input therefore does not prevent the migration, but ordinary
subsequent generation still requires it. Extra unknown owners declared at TARGET
and target peers declared at SOURCE refuse. Unrelated config roots retain their
disjointness and stable canonical ancestor-binding checks.

The command regenerates the **complete original source group**, including source
peers retained by a selected-owner move. Kotlin membership/bytes, lock modes,
profiles, allocations and occurrences must remain exact. New source features or
renames must be applied separately. Target settings still undergo ordinary
configuration/dependency/path validation. Explicit `runtime_package` retains
its existing same-root requirement, checked for both original declarations and actual post-move placement: selected source owners at TARGET, retained owners at SOURCE, and other configured peers at their declared roots. This action does not distribute a shared runtime between roots, even when original source declarations already name TARGET.

## State preservation and transactions

Complete source and target observations are compared again in readonly planning
and under the complete sorted two-root lock set. Any intervening valid manifest,
raw lock, code, mode or mtime change refuses with `E_OUTPUT_ROOT_MOVE_STALE`.
Missing, edited, malformed, foreign or unfinished state can fail its existing
storage diagnostic first. Both raw manifests participate in CAS, including
noncanonical but valid JSON formatting. Check and validate do not acquire locks,
create directories or write sidecars/stages.

The merged target manifest records the same-version union of owners, Kotlin
hashes and raw lock hashes. Existing target Kotlin/raw locks are never rewritten;
their bytes, modes and mtimes remain unchanged. Newly copied Kotlin/raw locks retain
the source's exact bytes, modes and mtimes, including valid noncanonical lock
serialization. Shared existing target Kotlin retains target attributes and is
never staged or rewritten; both inventories still participate in complete CAS. Unowned files remain untouched. A full move removes only original
source-owned files/locks/manifest; a selection retains original peer/shared
memberships. The changed manifests are deterministically serialized; their raw
JSON and attributes are not covered by the payload-preservation promise.

Every participating journal stages before any root applies and handles its
manifest last. Normal failures roll both roots back, including original raw
manifests and payload attributes. An obstructed rollback retains recovery
backups, attempts the other root and returns `E_RECOVERY_REQUIRED`. There is no
cross-root crash atomicity or automatic recovery. Existing cross-volume and
noncooperating-filesystem-mutation limits still apply.

## Reports and workflow

`output_root_migration.manifest_sha256` binds the original full source manifest.
A merge additionally reports `target_manifest_sha256` for the exact observed
target manifest and sorted `target_owners` for its original inventory. Sorted
`shared_target_files` identifies moved Kotlin paths already owned by target;
those paths retain target attributes. Existing
`files`/`name_locks` describe the distinct moved payload union; each file retains all original source owner references, including retained sharing peers. Selected moves report sorted `moved_owners`
and source `retained_owners`. Text exposes the same target hash/owners. Reports
omit absolute machine paths and do not bind a future command as an approval token.

```sh
openapi-kt generate --config openapi-kt.toml \
  --move-output-from generated-old --output generated-existing \
  --check --diagnostic-format json
# Move a and b together while retaining other original source owners.
openapi-kt generate --config openapi-kt.toml \
  --move-output-from generated-old --output generated-existing \
  --move-owner a --move-owner b --check --diagnostic-format json
openapi-kt generate --config openapi-kt.toml \
  --move-output-from generated-old --output generated-existing
```

A safe nonidentity check returns 3, validate returns 0, and apply returns 0.
Incompatible owned inventories return 4 / `E_OUTPUT_ROOT_MOVE_OCCUPIED`;
colliding handwritten destinations, stale/invalid state and metadata limits use
their existing diagnostics. Update the moved specs' `output_dir` values, drop
the one-shot flag, and confirm ordinary full config `--check` with normal inputs
available. Update consumer source directories and include shared Kotlin once.

[Full/selected profiles, raw payloads, exact budgets, CAS and every transaction
boundary](../internal/output/root_move_merge_test.go), [native locks/bindings and
obstructed rollback](../internal/output/root_move_merge_unix_test.go), [bounded
rollback model](../internal/output/root_move_merge_fuzz_test.go), [native config,
reports and post-move no-op](../internal/cli/root_move_merge_process_test.go) and
[native pre-input FIFO target CAS](../internal/cli/root_move_merge_unix_process_test.go)
define this O03/T01/T02 profile. Shared-path [payload/attribute and transaction checks](../internal/output/root_move_shared_merge_test.go),
[bounded rollback model](../internal/output/root_move_shared_merge_fuzz_test.go),
[native locks/recovery checks](../internal/output/root_move_shared_merge_unix_test.go),
[native Data/Plain profiles/reports/no-op](../internal/cli/root_move_shared_merge_process_test.go)
and [pre-input target FIFO CAS](../internal/cli/root_move_shared_merge_unix_process_test.go)
cover shared target memberships. Generated Kotlin, wire behavior and persisted
format versions do not change. Broader ownership lifecycle and platform
acceptance remain separate.
