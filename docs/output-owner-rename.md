# Named-owner rename

`generate` and `validate` support an invocation-only, repeatable
`--rename-owner OLD:NEW` with `--config FILE`. This transfers an existing
[version 2 or version 4 named owner](output-ownership.md) to a current configured
spec name in the same output root. It changes ownership metadata only: Kotlin
membership, source bytes, public classifiers and serializer descriptors must
remain unchanged. Version 2 remains version 2; version 4 remains version 4.

## Selection and invocation

`NEW` must be the selected current spec name. `OLD` must already own output in
that spec's effective configured output root. Both names use the case-sensitive
ASCII owner grammar: a letter followed by letters, digits, `_` or `-`.
`--rename-owner=OLD:NEW` is also accepted. Each old and each new name may occur
only once in the invocation.

- With `--spec NEW`, supply exactly one mapping whose target is `NEW`.
- Without `--spec`, map every current configured spec, including unchanged
  owners using identity mappings such as `events:events`.
- `OLD=NEW` is a no-op only after the same ownership, candidate and state
  validation succeeds. It does not bypass safety checks.

There is no persistent TOML rename key. The flag requires `--config` on
`generate` or `validate`; direct file/stdin commands cannot use it. It cannot be
combined with `--adopt-legacy`, `--adopt-locked-owner`, `--move-output-from`, `--reconcile` or `--name-lock-action`.

Change the configured spec name first, keeping its input, output root and
emission options consistent with the existing output. For an existing `catalog`
owner whose current configured name is now `inventory`:

```sh
openapi-kt generate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory --check --diagnostic-format json
openapi-kt validate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory --diagnostic-format json
openapi-kt generate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory
# Subsequent generation uses the current owner, without the one-shot mapping.
openapi-kt generate --config openapi-kt.toml --spec inventory --check
```

For full selection where `events` already has its current name:

```sh
openapi-kt generate --config openapi-kt.toml \
  --rename-owner catalog:inventory --rename-owner events:events \
  --check --diagnostic-format json
```

A completed nonidentity rename cannot replay the old mapping: the old owner no
longer exists. Ordinary generation with the new name is the next step.

## Exact preservation and rejection

Rename reads the manifest, every manifest-owned Kotlin file and every owned
name-lock payload in each participating root, including retained peers. All
owned files must be regular, present, safe and hash-matching; all metadata must
be valid and within the existing [name-lock limits](name-lock.md). Modified,
missing, malformed or unsupported state fails without repair or replacement.
Handwritten files remain protected. An unowned destination for the new owner's
dedicated name-lock filename is a collision, even if its bytes match.

Generation uses the current selected spec and the old owner's previous lock
state. The candidate must preserve the old owner's exact Kotlin paths and
bytes, its locked/unlocked mode and, when locked, its package, suffix,
qualification profile, allocations and original occurrences. Adding or removing
files, changing source bytes, growing allocations or moving occurrences is a
separate generation/lifecycle change and is rejected during rename. Existing
core profile/shape compatibility checks also continue to apply.

An unknown old owner fails. An occupied new owner is accepted only when it is
also a source in the complete selected mapping for the **same output root**.
This admits simultaneous chains and cycles, such as `alpha:beta, beta:alpha`.
Every selected new spec keeps its old source's input, package/options and lock
mode; all exact-candidate checks use the original inventory. An unselected
occupied target still fails, including under partial `--spec` selection.

For an existing alpha/beta pair whose current spec names have been exchanged:

```sh
openapi-kt generate --config openapi-kt.toml \
  --rename-owner alpha:beta --rename-owner beta:alpha --check
openapi-kt generate --config openapi-kt.toml \
  --rename-owner alpha:beta --rename-owner beta:alpha
# Use the current names and inputs after applying the one-shot mapping.
openapi-kt generate --config openapi-kt.toml --check
```

The complete mapping is applied simultaneously. Kotlin paths and shared
memberships follow the original source owner; dedicated raw lock payloads move
to the final target filename. An overwritten destination is staged once and
never also removed. Identical final memberships and raw payloads form a validated
no-op, preserving existing metadata bytes and modification times. Reusing an old
mapping is a new request against the current inventory, rather than an automatic
retry; distinct candidates refuse if the old source identity no longer matches.

Rename does not reconcile:
all other owners, their metadata and shared memberships remain, including
owners absent from the current configuration. Only the old identifier in a
selected membership is replaced by its new identifier; shared Kotlin files stay
on disk.

Only roots selected by the current effective configuration participate. Partial
`--spec` does not read unselected inputs or inspect their other output roots;
within its selected root it still validates all owned peers. Full rename retains
peers absent from the inventory rather than applying ordinary full-generation
reconciliation. There is no former-root discovery, output-root move within this rename action, version 1
adoption or direct version3-to-named-version4 migration. [Explicit complete root
move](output-root-move.md) is a separate same-version full-root workflow. Use the separate
[exact-match version1 adoption](ownership-migration.md), [locked-owner adoption](ownership-migration.md#locked-owner-adoption) or [disable/reset
lifecycle](name-lock.md#explicit-disablereset-lifecycle) for their documented
purposes.

## Read-only preflight and transactions

The initial read records exact manifest bytes and all owned lock bytes. Output
preflight rereads the complete owned state and compares those bytes; commit
repeats that comparison under the complete cooperative root lock set, acquired
nonblocking in canonical sorted order. A change to the manifest or any owned
lock, including a peer's payload, rejects the candidate. Valid intervening
ownership/state changes return `E_OWNER_RENAME_STALE`; invalid or modified state
can fail its storage validation first. Kotlin presence and hashes are checked
again. Rename does not automatically regenerate or retry after a concurrent
update.

`generate --check` and rename `validate` perform the same read-only candidate
preflight without acquiring writer locks or creating directories, sidecars or
stages. A check exits 3 for a safe metadata change and 0 for a validated no-op;
validation exits 0 for a safe candidate even when applying it would change
metadata. These reports describe a point-in-time observation, not a filesystem
snapshot or approval token binding a later apply.

Apply holds every participating root lock through application or rollback and
stages all changed roots before applying any. The journal contains only metadata
operations, with each manifest last. For a locked owner, its exact old payload
bytes are installed at
`.openapi-kt-names-<sha256(exact NEW identifier)>.json`; old filenames without
incoming locked ownership are removed, and version4's `owners`/`name_locks` keys are updated. Unlocked rename
changes only the manifest owner key. Kotlin files are never written or removed;
their bytes, modes and modification times remain unchanged. Identity does not
rewrite metadata. The persistent `.openapi-kt.lock` remains a cooperative writer
sidecar; apply can create a missing sidecar.

Busy writers refuse apply with `E_OUTPUT_BUSY`. Read-only preflight does not
acquire that lock and offers no busy-writer exclusion. [Retained transaction
stages](recovery-rejection.md) refuse read, check, validation and apply with
`E_RECOVERY_REQUIRED`, preserving evidence. Normal application failures roll
back metadata across participating roots, restoring old manifest and lock
bytes, modes and modification times when rollback succeeds. If restoration
fails, the writer still attempts the other roots, retains affected journals and
backups, and returns `E_RECOVERY_REQUIRED`. These are [normal-failure batch
guarantees](output-batches.md), without cross-file/cross-root crash atomicity or
automatic recovery.

## Reports and exit codes

JSON adds `owner_renames` to the existing report, with one entry per selected
mapping sorted by `new_owner`:

| Field | Meaning |
|---|---|
| `old_owner`, `new_owner` | Exact case-sensitive identifiers |
| `version` | Existing manifest version, 2 or 4 |
| `files` | Sorted unchanged Kotlin membership, each with relative `path` and lowercase SHA-256 `sha256` |
| `name_lock` | Present only for a locked owner; records the exact preserved payload `sha256`, `profile` and `symbols` |
| `name_lock.profile` | `package`, `suffix`, `qualified` |
| `name_lock.symbols` | Each records `role`, `shape`, public `classifier`, canonical emitted `descriptor` and original `occurrences` |
| `impact` | Metadata-only transfer, retained peers/shared memberships and guidance to keep the new spec name |

Text reports expose the same owner, version, file-hash, optional lock/profile/
symbol and impact data. Metadata is excluded from the top-level Kotlin `files`
list. Relative paths may repeat across roots, as in ordinary config reports;
absolute output roots are not added. Existing warnings and warning-denial policy
still apply before lock acquisition or mutation.

| Exit | Rename behavior |
|---|---|
| 0 | Successful apply, safe rename validation or validated check no-op |
| 1 | Input/config/core generation or warning-policy error, including core `E_NAME_LOCK_PROFILE` |
| 2 | Parser/usage error: malformed/repeated names, missing config or incompatible flags |
| 3 | Safe `generate --check` metadata difference |
| 4 | Storage or rename refusal: invalid selection/owners, changed candidate/mode/state, stale bytes, collision, missing/modified output, busy writer or recovery evidence |

Rename-specific refusals include `E_OWNER_RENAME_INVALID`,
`E_OWNER_RENAME_CHANGED` and `E_OWNER_RENAME_STALE`; existing storage and
name-lock diagnostics remain applicable. Keep the output and metadata intact
when reviewing a refusal.

Verification surfaces are [membership/state/identity and fault-boundary
contracts](../internal/output/owner_rename_test.go), [Unix complete-lock-set and
CAS contracts](../internal/output/owner_rename_unix_test.go), [bounded state
fuzzing](../internal/output/owner_rename_fuzz_test.go), [native CLI selection,
reports and source parity](../internal/cli/owner_rename_process_test.go),
[simultaneous permutation and overwrite rollback contracts](../internal/output/owner_permutation_test.go),
[native permutation reports/replay](../internal/cli/owner_permutation_process_test.go),
[full TOML and namespace consumer source parity](../internal/cli/owner_rename_namespace_process_test.go),
and [native Unix busy/CAS contracts](../internal/cli/owner_rename_unix_process_test.go).
Test entry points do not establish candidate-wide or cross-platform runtime
acceptance, crash recovery or arbitrary filesystem fault coverage.
