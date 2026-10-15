# Explicit ownership adoption

Direct unlocked version1 adoption and locked version3 adoption have distinct entry points. The locked workflow is documented under [locked owner adoption](#locked-owner-adoption); the existing version1 rules below are unchanged.

## Version 1 ownership migration

Direct `--input` generation uses a single-owner version 1 manifest. [Configuration](configuration.md) generation uses version 2 named owners. It refuses version 1 output by default (`E_MANIFEST_OWNER_MIGRATION`), rather than inferring which spec owns old files.

An explicit **exact-match, metadata-only** migration is available:

```sh
openapi-kt generate --config openapi-kt.toml --spec catalog --adopt-legacy --check
# After reviewing the selected owner and authorizing migration:
openapi-kt generate --config openapi-kt.toml --spec catalog --adopt-legacy
```

`--adopt-legacy` is a bare, nonrepeatable generate flag. It requires `--config` and an explicit `--spec NAME`; it is unavailable for direct-input generation, validate, download or empty reconciliation. It means the selected spec owns the **entire** old manifest, not just a subset. Agents should obtain the user's explicit migration intent before applying it to an existing tree.

## Preconditions

Use the same local document and effective generation options as the old direct command. The selected spec's generated relative paths and content hashes must **exactly equal** the legacy manifest's complete file set. Each existing owned file must still be a regular file with the recorded content hash, under safe parents. No unowned files are adopted. Wrong suffix/package/options/schema/path sets fail with `E_MANIFEST_OWNER_MISMATCH`; modified files fail with `E_OUTPUT_MODIFIED`, missing files with `E_OUTPUT_MISSING`. Corrupt/unsupported manifests and recovery stages remain errors; the flag does not permit resetting them.

For a version 1 tree, check returns 3 for the proposed ownership metadata change, even if Kotlin sources are identical. It does not acquire locks, create directories or write files. Successful migration returns 0, changes only the manifest's version/membership, and preserves every Kotlin file's bytes, modification time and mode. Handwritten files and existing coordination files remain unchanged. The new manifest lists the selected owner for every migrated path; unselected configured specs are not generated or assigned those files.

## After migration

Ordinary `generate --config ... --spec ...` and check can now use named ownership. Rerunning the same command with `--adopt-legacy` on version 2 is idempotent for unchanged input/options; the flag does not keep constraining later version 2 updates to the original version 1 hashes. On a fresh root it performs ordinary named generation, still refusing unowned collisions. Direct-input version 1 writers continue to refuse version 2 manifests.

Preserve the original options for migration first, then perform intentional model/API changes through normal configuration generation. This workflow does not rename/move owners or roots, split/merge a legacy manifest among multiple specs, repair missing files, or implement arbitrary manifest-version upgrades. It does not remove obsolete output roots.

The migration uses the existing [named-owner batch transaction](output-batches.md). Legacy membership is synthesized only in memory after exact validation; the previous manifest remains version 1 until commit. Checks and lock-time replanning repeat validation. Normal failure restores the original version 1 manifest bytes and modification time; a later peer-root failure also rolls back an earlier metadata adoption. Failed rollback retains the original manifest backup/journal and blocks further generation/check with `E_RECOVERY_REQUIRED`. This is not crash atomicity or automatic crash recovery.

Verification: [metadata-only/strict refusal/rollback tests](../internal/output/adoption_test.go), [mixed-version batch and retained version 1 backup](../internal/output/adoption_batch_test.go), [bounded adoption fuzz](../internal/output/adoption_fuzz_test.go), actual native [adoption/check/idempotency](../internal/cli/config_adoption_process_test.go), [argument/mismatch/error preservation](../internal/cli/config_adoption_guards_process_test.go), and [Unix held-lock/check behavior](../internal/cli/config_adoption_unix_process_test.go).

## Locked owner adoption

An existing complete direct **version3** root can explicitly become one named **version4** owner in the **same output root**:

```sh
openapi-kt generate --config openapi-kt.toml --spec catalog \
  --adopt-locked-owner --check --diagnostic-format json
openapi-kt validate --config openapi-kt.toml --spec catalog \
  --adopt-locked-owner --diagnostic-format json
openapi-kt generate --config openapi-kt.toml --spec catalog --adopt-locked-owner
# Subsequent management uses this named owner.
openapi-kt generate --config openapi-kt.toml --spec catalog --check
```

`--adopt-locked-owner` is invocation-only, bare and nonrepeatable on generate or validate. It requires `--config`, explicit `--spec NAME`, an inventory containing exactly that one spec, and effective `hoist_shared_shapes = true` and `lock_hoisted_names = true`. That spec claims the entire original Kotlin inventory. Keep its local input, effective options, output root, package, suffix and qualification profile consistent with the original direct command. There is no TOML action key. Combine neither `--adopt-legacy`, `--name-lock-action`, `--rename-owner`, `--move-output-from` nor `--reconcile` with this action. Direct input/stdin, fresh/unlocked roots, partial inventories, multi-owner additions, owner splitting and general format upgrades are outside this profile.

This separate entry point preserves the existing `--adopt-legacy` acceptance/refusal behavior: version1 migration still has its generate-only, fresh-root and subsequent normal-update semantics. Locked adoption instead requires existing valid state, supports read-only validation, and constrains every invocation to exact preservation.

### Exact preservation and source binding

Before reading the selected input, the adapter captures the original selected path, its canonical root and directory identity, exact manifest bytes, every manifest-owned Kotlin file's bytes and the owned raw lock payload. It validates regular files, safe parents, complete membership, content hashes and lock syntax/limits. This action adds a **64 MiB aggregate old-Kotlin capture limit** (`E_OUTPUT_LIMIT`), independent of the existing per-file limit of the same size. It applies to v3 adoption and v4 no-op reads. A size-only pass checks all owned Kotlin paths, regular-file sizes and the complete sum before allocating any Kotlin payload capture; actual reads separately recheck the remaining budget and hashes, so file growth after preflight cannot be accepted above the bound. Resource rejection precedes content hashing. Manifest/raw-lock reads retain their separate existing limits; all core/name-lock/configuration budgets remain unchanged. This bounds accepted capture bytes, not process RSS: repeated CAS snapshots, metadata, generated artifacts and transient buffers also consume memory. It is not benchmark or speedup evidence. The regenerated candidate must have exactly the same sorted Kotlin paths and bytes, locking mode, stored profile, allocations and original occurrences. Classifier/descriptor identity and wire behavior follow those unchanged sources and lock semantics; no allocation reset or source rewriting is performed. Input annotations or option differences without any effect on those persisted/generated identities are not a separately stored input-byte history.

The reader refuses missing/edited files or state, malformed/unsupported metadata, symlinks at the root or managed paths/parents, unfinished recovery stages and an occupied named destination, even if its bytes match. Existing ancestor aliases resolve normally, but rebinding an ancestor after capture or replacing the root directory rejects the original binding. A different output override is never implicit migration: a fresh destination refuses; an independently existing destination would need its own complete source state and exact candidate match. There is no former-root discovery or movement. An explicit `--output` may fill an omitted or empty configured `output_dir` only by selecting that existing same-root source; all adoption state/candidate checks still apply. The separate `--move-output-from` action continues to require every spec's nonempty declared root before the destination override and before state/input IO.

Read-only output preflight rereads the whole source and compares exact manifest, Kotlin and raw lock bytes to that pre-input capture. Apply repeats it under the complete sorted cooperative lock set before staging. A valid intervening source change returns `E_LOCKED_OWNER_ADOPTION_STALE`; invalid/modified state can fail its existing storage diagnostic first. The adapter does not retry or regenerate automatically.

### Transaction and downstream impact

Only ownership metadata changes. The exact bytes of `.openapi-kt-names.json` move to `.openapi-kt-names-<sha256(exact NAME)>.json`, including valid noncanonical whitespace/key order. The version4 manifest assigns all original Kotlin paths to `NAME` and maps only that owner's lock to the raw payload SHA256. Kotlin files are never journaled, written or removed; their bytes, modes and modification times remain unchanged. Similar unowned metadata, handwritten files and other roots remain intact. The old anonymous payload is removed only because version3 explicitly owned it.

Apply uses the existing [batch transaction](output-batches.md): locks held through application/rollback, all staging before destination mutation, manifest last. Normal failure restores the old version3 manifest and anonymous lock bytes/modes/mtimes; Kotlin was untouched. Obstructed rollback still attempts remaining restoration, retains the affected journal/backups and returns `E_RECOVERY_REQUIRED`. Reads/check/validate/apply refuse retained stages. These are normal-failure guarantees, without crash atomicity, hostile-filesystem exclusion or automatic recovery. Preserve evidence rather than deleting locks/manifests/stages to force a fresh allocation.

After success, use configuration with the same named owner for ordinary generation/check/validation. Ordinary direct version3 invocation refuses version4; it cannot implicitly keep managing that tree. Exact same-owner, complete single locked version4 state can replay this action as a validated no-op, preserving even raw noncanonical lock bytes. Changing membership/source bytes or allocations during replay still refuses; intentional later updates belong to ordinary named generation. Existing ordinary named generation retains its canonical lock-writing behavior. Adoption does not rename owners, disable/reset state or reconcile peers.

### Reports and verification

JSON adds `locked_owner_adoptions`, one entry with `owner`, `before_version`, `after_version`, `manifest_before_sha256`, `manifest_after_sha256`, sorted unchanged `files[{path,sha256}]`, `name_lock{before_sha256,after_sha256,profile,symbols}` and `impact`. Profile has package/suffix/qualified; each symbol has role/shape/public classifier/canonical emitted descriptor/original occurrences. The proposed manifest hash binds the exact bytes staged on v3 adoption; v4 no-op reports its unchanged raw manifest hash. Text exposes the same data. Absolute roots are excluded and top-level files remain Kotlin-only.

Check and validate use the same candidate/profile/source binding and complete read-only preflight, creating no directories, locks or stages and preserving file/directory modification times. Check exits 3 for safe v3→v4 metadata differences, 0 for a validated v4 no-op; safe validation exits 0. Read-only results are point-in-time observations, not concurrent snapshots or approval tokens for later apply. Busy writers exclude apply (`E_OUTPUT_BUSY`), while read-only preflight acquires no writer lock. Usage failures are exit2, core/profile/input parsing/warning failures exit1, and input IO/storage/adoption refusals exit4, including `E_LOCKED_OWNER_ADOPTION_INVALID`, `E_LOCKED_OWNER_ADOPTION_CHANGED` and `E_LOCKED_OWNER_ADOPTION_STALE`.

[Output preservation/fault contracts](../internal/output/locked_owner_adoption_test.go), [Unix binding/CAS/complete lock set](../internal/output/locked_owner_adoption_unix_test.go), [bounded fuzz](../internal/output/locked_owner_adoption_fuzz_test.go), [native reports/no-op](../internal/cli/locked_owner_adoption_process_test.go), [refusal/read-only guards](../internal/cli/locked_owner_adoption_guards_process_test.go), [native FIFO CAS/busy](../internal/cli/locked_owner_adoption_unix_process_test.go), [native explicit-root versus move-declaration guards](../internal/cli/locked_owner_adoption_declared_root_process_test.go), [current consumer byte controls](../internal/cli/locked_owner_adoption_parity_process_test.go) and [actual merged descriptors/YAML](../internal/cli/locked_owner_adoption_descriptor_process_test.go) cover this bounded D09/O03/T01/T02 increment. Native byte controls and unchanged consumer files are source/API preservation evidence, not fresh Kotlin compilation/runtime or cross-platform runtime proof; wider lifecycle support remains separate.
