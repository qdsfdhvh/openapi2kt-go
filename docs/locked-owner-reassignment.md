# Reassign locked owner memberships

`generate|validate --config FILE --reassign-locked-owners` explicitly redistributes
one existing root's locked named ownership while preserving its **complete
original Kotlin and historical allocation names**. Use the full desired owner
inventory without `--spec`. New owner IDs and removal of old IDs are allowed
when the desired owners still cover every original Kotlin file and every
original allocation occurrence.

For example, move an existing client's methods, generated response files and
hoisted locks from `api` to an existing `models` owner while `api` retains its
ordinary component. Ordinary generation refuses removed locked occurrences;
resetting names can rename historical public classifiers. This action projects
the original complete lock inventory into the desired owners and changes only
ownership/lock metadata. No Kotlin file is written or removed.

## Eligibility and selection

The existing root must be nonempty version4 named output, with every old owner
locked. All old locks must have the same package, suffix and qualified-name
profile. A role/shape must have one original public name, and a role/source
occurrence must denote one original allocation across owners. Conflicting
names, profiles or occurrences refuse before input generation.

All desired specs must use that profile, hoisting and locking, and one concrete
existing output root. Configuration precedence, path resolution, dependency
validation and ordinary generation-option override restrictions apply. The
configuration describes the **entire desired inventory**, including unchanged
peers; omitted old owners are removed only after complete preservation checks.
Each desired owner needs nonempty generated Kotlin membership. The action does
not discover other roots or infer omitted ownership.

The action is invocation-only, takes no value and cannot combine with `--spec`,
`--reconcile`, adoption/donor flags, owner rename, root move, or lock reset/disable.
There is no TOML action key. Ordinary [reconciliation](configuration.md#ownership-and-failure-safety)
already handles removal of omitted owners and no-longer-owned files; [owner
rename](output-owner-rename.md) and [donor adoption](ownership-migration.md#add-an-owner-to-an-existing-locked-inventory)
retain their separate contracts. Use reassignment when historical locked
memberships must move without resetting names or changing any Kotlin.

## Preservation and limits

Before any desired input is read, the command captures the complete original
raw manifest, all owned Kotlin, all raw locks, file modes/mtimes, canonical root
binding and physical root identity. Missing, edited, foreign, unsafe or
unfinished state refuses. Unowned files remain untouched. All old owners
participate in CAS, including those omitted from the desired configuration.

The original allocation union must itself fit existing single-lock limits:
1 MiB encoded lock, 4096 allocations and 16384 occurrences. Shared identical
allocations/occurrences are counted once in that union. Existing limits also
apply to at most 256 old/desired owners, 16 MiB aggregate old and actual resulting
raw locks, 64 MiB original Kotlin, 64 MiB aggregate generated candidate Kotlin,
and ownership metadata/parser budgets. Candidate shared Kotlin counts once per
candidate owner for the aggregate budget. Actual original reads consume the
remaining source budget; valid separate inventories do not bypass union limits.
No cap is enlarged for reassignment.

Each candidate uses the existing explicit lock projection: original names and
retained singleton allocations remain. New/changed allocations or occurrences
refuse. Every candidate file must match an exact original relative path and
bytes, and the complete candidate Kotlin union must equal the original inventory.
The complete projected allocation/occurrence union must also equal the original;
leaving an owner, file or occurrence unassigned refuses. These checks preserve
original Kotlin call sites, descriptors and wire behavior through exact source
preservation rather than a general API equivalence analysis.

Unchanged semantic locks retain their original raw bytes. The same owner's
payload is preferred; otherwise an unchanged semantic donor is chosen in sorted
old-owner order. Exact copied donor payloads preserve donor mode/mtime. Newly
projected metadata uses canonical lock serialization and ordinary staging
attributes. Removed owners lose only their manifest-owned lock files. Kotlin
and unowned files remain in place with original bytes/modes/mtimes. Manifest
version4 and lock version1 remain unchanged.

## Planning, transactions and reports

Check and validate perform the same readonly candidate/storage checks without
locks, directories, sidecars or stages. Apply rechecks complete raw state and
attributes, canonical binding and physical identity under the cooperative root
lock. Any intervening valid state/attribute change returns
`E_LOCKED_OWNER_REASSIGN_STALE`; missing/edited/invalid state may fail its existing
storage diagnostic first. There is no automatic retry or rebase onto newer state.

Only metadata enters the journal, and the manifest is last. Normal failures
restore original raw metadata, modes and mtimes. Obstructed rollback preserves
backups and returns `E_RECOVERY_REQUIRED`; no automatic recovery or crash
atomicity is promised. Cooperative locks retain their existing limits against
noncooperating filesystem mutation. Warning policy applies to the entire desired
generated inventory before any metadata plan or write.

JSON adds `locked_owner_reassignment`:

| Field | Meaning |
|---|---|
| `version` | Preserved manifest version4 |
| `manifest_before_sha256`, `manifest_after_sha256` | Original/result raw manifest hashes |
| `before_owners`, `after_owners` | Sorted complete owner inventories; additions/removals are explicit |
| `files` | Complete unchanged Kotlin path/hash inventory |
| `owners` | Each old/new owner's before/after file memberships and lock hashes, plus resulting public symbols |
| `source_symbols` | Complete original union with role/shape/classifier/descriptor/original occurrences |
| `impact` | Metadata-only preservation and transaction scope |

Text exposes the same file hashes, owner memberships, lock hashes and public
symbols. Absolute machine paths are omitted. Reports are point-in-time results,
not an approval token for a future command. Exact replay retains noncanonical
raw manifest/lock bytes and is a validated no-op.

```sh
openapi-kt generate --config desired-owners.toml \
  --reassign-locked-owners --check --diagnostic-format json
openapi-kt validate --config desired-owners.toml \
  --reassign-locked-owners --diagnostic-format json
openapi-kt generate --config desired-owners.toml --reassign-locked-owners
openapi-kt generate --config desired-owners.toml --check
```

Safe changed check returns 3; validate/apply and exact replay return 0. Usage
conflicts return 2, input/core/warning-policy failures return 1, and storage or
preservation failures return 4. Specific safety diagnostics are
`E_LOCKED_OWNER_REASSIGN_INVALID`, `E_LOCKED_OWNER_REASSIGN_CHANGED` and
`E_LOCKED_OWNER_REASSIGN_STALE`; original profile/conflict/limit/storage diagnostics
also apply. Keep the desired configuration for subsequent ordinary generation
and drop the one-shot action flag.

[Metadata-only preservation/coverage/replay](../internal/output/locked_owner_reassignment_test.go),
[original union and desired-owner limits](../internal/output/locked_owner_reassignment_limits_test.go),
[whole omitted-peer CAS and original authority](../internal/output/locked_owner_reassignment_guards_test.go),
[native locks/bindings/root identity/recovery](../internal/output/locked_owner_reassignment_unix_test.go),
[bounded rollback model](../internal/output/locked_owner_reassignment_fuzz_test.go),
[native historical names/profiles/owner changes/reports](../internal/cli/locked_owner_reassignment_process_test.go)
and [native pre-input FIFO CAS](../internal/cli/locked_owner_reassignment_unix_process_test.go)
cover this O03/T01/T02 profile. Mixed/unlocked or incompatible-profile inventories,
incomplete adoption, wider lifecycle migrations and platform-wide acceptance
remain separate.
