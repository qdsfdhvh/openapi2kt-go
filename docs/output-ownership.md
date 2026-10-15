# Output ownership

Unlocked direct native CLI `--input` commands use the existing **single-owner version 1** manifest. Opt-in [name-lock state](name-lock.md) adds direct version3/configured version4 metadata ownership; it remains outside the Kotlin path map and commits with generated files. The [local configuration envelope](configuration.md) now connects `--config`/`--spec` generation to version 2 named owners. [Batch commits across disjoint roots](output-batches.md) provide normal-failure rollback; cross-root crash atomicity and automatic migration are not available. [Explicit output-root move](output-root-move.md) copies one complete existing v1–v4 owned root to an explicitly selected target with raw bytes, file attributes and unchanged ownership; full config must contain every original owner in one root. [Shared runtime packages](runtime-package.md) combine needed Patch/bigint helper files through identical-byte owner membership. Explicit [named-owner rename](output-owner-rename.md) transfers an existing version2/4 owner to a selected current spec in the same root while preserving exact Kotlin membership and bytes.

An internal single-root named-owner adapter now provides a foundation for that workflow. `internal/output.WriteOwners` is not a stable public Go API. It accepts the complete active owner inventory, generated artifacts for selected owners and an explicit reconcile decision. Partial updates preserve all unselected owners, even if absent from that invocation's active inventory. Reconciliation removes only owners absent from the active inventory. An explicitly selected empty artifact set removes that owner's membership; explicit empty reconciliation cleans manifest-owned files without touching handwritten files.

## Portable paths and cooperative locks

Generated artifact paths and manifest keys must already be slash-separated relative Kotlin file paths on every host. Backslashes, drive/stream separators and traversal components are rejected before output mutation; adapters do not normalize invalid artifact paths into valid ones. Native filesystem syntax remains valid for the separately selected output directory. Callers constructing artifacts on Windows must use `/` in artifact paths rather than native `\` separators, consistent with the public `GeneratedFile.Path` contract.

The persistent `.openapi-kt.lock` is a cooperative writer sidecar, not generated payload. Unix uses `flock`; Windows uses a nonblocking exclusive `LockFileEx` byte-zero lock. Checks do not acquire this writer lock. Windows byte locks can also prevent reads through a different handle, so tools inspecting output state must not assume a held sidecar is readable, even when empty. This does not relax checks on generated files, manifests or name-lock metadata, and does not change writer exclusion or rollback behavior.

## Version 2 membership

The named-owner adapter uses a distinct manifest format:

```json
{
  "version": 2,
  "files": {"org/example/Runtime.kt": "<lowercase SHA-256>"},
  "owners": {
    "alpha": ["org/example/Runtime.kt"],
    "beta": ["org/example/Runtime.kt"]
  }
}
```

The illustrative hash placeholder is not valid manifest data. Actual hashes are 64 lowercase hexadecimal characters. Owner identifiers are case-sensitive ASCII identifiers beginning with a letter, followed by letters, digits, `_` or `-`. Each owner's nonempty path array is strictly sorted, with no duplicate paths; every file must have at least one owner. Paths, content hashes, case-insensitive file collisions and duplicate JSON keys are validated. JSON object key order does not change ownership semantics.

Two owners may share the exact same path only with identical bytes. Changing a selected owner's shared file while another retained owner requires its old content fails rather than overwriting the other owner. A file is removed only after its final owner reference disappears. Missing retained files fail with `E_OUTPUT_MISSING`; regenerate the owning spec before a partial update. Modified files, unsafe paths, unfinished recovery stages and unowned destinations remain protected.

## Transactions and migration boundaries

[Interrupted transactions](recovery-rejection.md) are detected and rejected with retained journals/backups across direct/configured/check and naming-lock paths. Automatic recovery is not available.

The adapter reuses the existing journal/staging/backup transaction and cooperative output-root lock. Ownership metadata commits and rolls back together with Kotlin files. It recomputes retained ownership under the lock, preventing a stale pre-lock plan from losing an intervening owner's update. Check is read-only and does not create directories or acquire locks; unchanged generation preserves file bytes and modification times. The single-root adapter is not a cross-root transaction; the separate batch adapter provides normal-failure multi-root rollback, not cross-file/cross-root crash atomicity.

Directory checks are reused within one planning pass and, separately, within locked transaction application. Every new plan, including the lock-time replan, checks directories afresh. Individual destination regular-file, ownership and content-hash checks remain. Private staging files may be written concurrently by at most eight workers, capped by `GOMAXPROCS`; all workers finish before cleanup or application begins. Destination changes still follow journal order with the manifest last, and rollback still restores prior files and removes only directories created by the transaction. These optimizations do not provide a filesystem snapshot against external concurrent changes.

Ordinary writers continue to reject version3/4. Explicit [disable/reset](name-lock.md#explicit-disablereset-lifecycle) actions validate existing state and report public classifier/descriptor/file changes before using the same CAS/backup transaction; direct disable alone transitions version3 to version1, while named disable retains version4. Explicit [owner rename](output-owner-rename.md) validates every owned file and lock in selected roots, uses exact manifest and all owned lock bytes for CAS in preflight and under the complete sorted lock set, and journals metadata only with the manifest last. Rename retains all peers and shared memberships without reconciliation, including owners absent from the current config. Keep the manifest and metadata intact during review.

The separate [locked-owner adoption](ownership-migration.md#locked-owner-adoption) action transfers a complete existing direct v3 root to one configured v4 owner in the same root. It binds canonical root identity plus exact manifest/all Kotlin/raw lock before input IO, repeats CAS in preflight and under sorted locks, and journals only the owned raw lock transfer and manifest last. Kotlin bytes/modes/mtimes and foreign metadata remain unchanged; subsequent management requires the named config owner. Fresh roots, implicit adoption, multi-owner additions and reconciliation are excluded.

Version 1 generation remains unchanged. Named-owner updates refuse to automatically adopt a version 1 tree (`E_MANIFEST_OWNER_MIGRATION`). The legacy writer refuses version 2 manifests. Malformed or unsupported manifests are not silently reset. [Explicit exact-match metadata-only adoption](ownership-migration.md) is available using `generate --config FILE --spec NAME --adopt-legacy`. Do not edit a manifest's version to bypass these safeguards.

The [owner tests](../internal/output/owners_test.go), [rollback/membership guards](../internal/output/owners_guards_test.go), [path guards](../internal/output/owners_paths_test.go), [Unix locking tests](../internal/output/owners_unix_test.go) and [state-transition fuzz model](../internal/output/owners_fuzz_test.go) verify shared/partial ownership, last-owner cleanup, empty reconcile, no-op/check, modified and handwritten file protection, strict versions, injected rollback and lock-time replanning. Platform runtime support and full multi-spec CLI/configuration are separate deliverables.
