# Explicit output-root move

`generate` and `validate` accept the one-shot `--move-output-from SOURCE` with an
explicit `--output TARGET`. Both paths identify concrete roots relative to the
invocation's working directory. The command moves **the entire manifest-owned
root** by default, keeping its existing version and exact Kotlin/metadata bytes. Configuration `--spec NAME` instead moves that named owner while retaining peers; see [selected-owner moves](#selected-owner-moves). Changing
`output_dir` or `--output` alone continues to select ordinary generation; it
never discovers or cleans a former root. Same-version v2/v4 named targets can also use the [owned-target merge profile](output-root-merge.md), with disjoint moved owner names and exact byte equality for shared Kotlin paths.

## Selection and input binding

Use the original input and emission options, including package, suffix,
validation, model style, merging, hoisting and locking. Generation is performed
in memory as an exact source control before any mutation. Kotlin membership and
bytes must match the source, and each existing lock must retain its mode,
profile, allocations and original occurrences. A new allocation, source change
or public/wire rename is rejected; perform it through a separate workflow.

| Source manifest | Accepted invocation |
|---|---|
| v1 | Direct `--input` with unchanged options and locking off |
| v3 | Direct `--input` with the stored hoist/locking profile |
| v2 | `--config` containing the complete original named-owner inventory, locking off |
| v4 | `--config` containing every original owner and its existing locked/unlocked mode |

Configuration requires a nonempty complete source inventory and one declared
root for that inventory; unrelated disjoint groups may remain in the same config. Every spec's effective `output_dir`, after defaults/spec overlay and before
the CLI target override, must be nonempty. Omission and explicit empty strings,
including an empty spec overriding a nonempty default, fail with
`E_OUTPUT_ROOT_MOVE_INVALID` (exit 4) before owned-state or input reads. TOML's
empty-value semantics and ordinary output overrides remain unchanged.
That declared `output_dir` must refer to the explicit source or target;
`--output TARGET` explicitly selects the final destination. For this action
only, that output override is allowed for a full multi-spec inventory. Other
emission overrides keep their single-selected-spec restriction. Config inputs
and `output_dir` retain config-directory resolution; the two CLI root arguments
use invocation-directory resolution. The source group means **exactly all existing
manifest owners with the same names**, including peers no longer in a reduced
configuration. Add those peers back with their original inputs/options first.
Shared paths retain every owner reference and move once with identical bytes.

Empty reconcile, incomplete source inventories,
missing/extra/renamed peers, `--rename-owner`, `--adopt-legacy`, `--adopt-locked-owner` and
`--name-lock-action` are outside this action. Incompatible flags fail with usage
exit 2; unsupported inventories fail closed with exit 4. This bounded action
provides no arbitrary multi-owner subset, merging of owner identities, manifest
adoption, v3-to-v4 conversion, historical-root search or metadata repair.
There is no persistent TOML move key.

```sh
# Direct v1 example; keep the original generation options.
openapi-kt generate --input openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new \
  --check --diagnostic-format json
openapi-kt validate --input openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new \
  --diagnostic-format json
openapi-kt generate --input openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new
openapi-kt generate --input openapi.json --package-name org.example.api \
  --output generated-new --check

# Full same-root config inventory; retain all original names/options.
openapi-kt generate --config openapi-kt.toml \
  --move-output-from generated-old --output generated-new \
  --check --diagnostic-format json
```

After a successful nonidentity move, keep the target in the configuration or
ordinary direct command and update the consumer's generated source directory.
Drop the one-shot source flag. Replaying a completed move fails because the
source no longer owns a manifest. Selecting exactly the same explicit source
and target is a fully validated no-op; it still verifies input and ownership.
Different paths aliasing the same canonical root are rejected.

## Source, destination and concurrency safety

The read-only initial observation verifies the existing strict manifest in its
legal v1/v2/v3/v4 meaning, every owned Kotlin file and every owned lock payload.
Files must be safe relative paths, regular, present and hash-matching; metadata
must satisfy existing [name-lock limits](name-lock.md). Unsupported formats,
missing/edited state, invalid code, foreign root-level `.openapi-kt-names*`
metadata and [unfinished stages](recovery-rejection.md) refuse the operation.
Unowned source files are never adopted or removed.

For the empty-target profile, the target must have no output manifest or foreign name metadata, even if it
appears identical or would describe owned files. Every destination for a moved
payload must be absent: identical unowned Kotlin also collides. Other handwritten
files and directories stay intact. No root is recursively cleared or deleted;
source cleanup removes only the reverified owned Kotlin, owned lock payloads and
manifest. Existing source directories and the cooperative sidecar remain.

Both roots use the existing canonical ancestor-resolution contract. The selected
root itself, payloads and their parents inside the root cannot be symlinks.
Parent traversal components, aliases, case-ambiguous roots and ancestor/descendant
overlap refuse planning. Existing ancestors above the selected root resolve
normally; an alias binding that changes before the locked plan is rejected.
Cooperative locks are not protection against arbitrary noncooperating filesystem
mutation or hostile TOCTOU changes.

The initial observation also captures source, target and declared configuration
root canonical bindings before input generation. Alias rebinding during input
reads refuses check, validation and apply rather than selecting another target.

Apply acquires the **complete canonical sorted root lock set**, nonblocking,
and holds it through application or rollback. Busy source or target returns
`E_OUTPUT_BUSY`. Inside the complete lock set it rechecks source bytes and
attributes, exact raw manifest and all owned lock bytes against the initial
observation; target absence/collisions and recovery state are checked again, together with all captured root bindings. Valid intervening source changes return `E_OUTPUT_ROOT_MOVE_STALE`; edited or
invalid state may fail its existing storage validation first. No automatic retry
or regeneration occurs.

Every changed root stages its entire journal on its own filesystem before any
root applies. Each journal handles its manifest last: target installs the
original raw manifest; source removes its owned manifest. Kotlin and lock state
are copied byte-for-byte with regular file permissions and modification times;
there is no cross-root rename dependency. Access times, directory timestamps,
ACLs, extended attributes, hard-link identity and filesystem-specific metadata
are outside this contract. The native tests exercise same-filesystem roots;
different-volume and other-host runtime acceptance require separate evidence.

Normal failures reverse all participating transactions and restore previous
code, metadata, ownership, file modes and mtimes when rollback succeeds. Existing
coordination sidecars remain unchanged; apply can create a missing target root,
parent directory or sidecar before a later failure, as in ordinary batches.
Rollback obstruction retains the affected journal/backups, still attempts the
other root and returns `E_RECOVERY_REQUIRED`. Process/OS/power loss between root
applications can leave different states. There is **no cross-root crash
atomicity, durable coordinator or automatic recover command**. Preserve residual
journals and backups before reviewing manual restoration.

## Reports and exit codes

Check and move validation run the same read-only candidate and output preflight:
no locks, directories, stages, sidecars or writes. Busy-writer exclusion belongs
to apply. These are point-in-time observations, not snapshots or an approval token
binding a later command.

JSON adds `output_root_migration`:

| Field | Meaning |
|---|---|
| `version`, `identity` | Preserved manifest version and whether this is the validated same-root no-op |
| `manifest_sha256` | Hash of the exact raw original full source manifest bytes |
| `target_manifest_sha256`, `target_owners` | Present for an owned-target merge: exact observed target manifest hash and sorted original target owner names |
| `shared_target_files` | Shared moved Kotlin paths retaining their existing target bytes, modes and mtimes |
| `files` | Sorted distinct Kotlin paths with `before`, `after`, preserved `sha256`, and sorted original `owners` |
| `name_locks` | Sorted owned payloads with optional direct/named `owner`, relative `path`, exact `sha256`, `profile` and `symbols` |
| `symbols` | Complete owned hoist lock inventory: role, shape, public classifier, emitted canonical descriptor and original occurrences |
| `impact` | Source-directory integration guidance, preservation scope and transaction limits |

`before`/`after` paths are relative to the explicitly chosen source/target and
remain equal. Direct files have an empty owner array. Text exposes the same
hashes, mappings and lock classifiers/descriptors. Top-level `files` remains
Kotlin-only; absolute machine paths are omitted. The hoist inventory is bounded
by the persisted lock; it is not a general API/ABI extractor or wire-equivalence
analysis. Exact source bytes provide the preservation control.

| Exit | Meaning |
|---|---|
| 0 | Successful move, safe validation or validated identity check |
| 1 | Input/config/core or warning-policy failure, including incompatible stored profile |
| 2 | Usage error, missing concrete root or incompatible flags |
| 3 | Safe nonidentity `generate --check` difference |
| 4 | Storage/migration safety refusal or transaction failure |

Move-specific refusals are `E_OUTPUT_ROOT_MOVE_INVALID`,
`E_OUTPUT_ROOT_MOVE_CHANGED`, `E_OUTPUT_ROOT_MOVE_OCCUPIED` and
`E_OUTPUT_ROOT_MOVE_STALE`; existing storage, busy and recovery diagnostics
remain applicable. This implements one D09/O03/T01/T02 increment; v3-to-v4 and
other lifecycle migration profiles remain open.

Verification surfaces include [v1–v4 preservation and fault boundaries](../internal/output/root_move_test.go),
[sorted locks, paths and retained evidence](../internal/output/root_move_unix_test.go),
[bounded rollback model](../internal/output/root_move_fuzz_test.go),
[native selection, reports and no-op](../internal/cli/root_move_process_test.go),
[declared inventory refusal and ordinary override control](../internal/cli/root_move_declared_roots_process_test.go),
[native FIFO CAS](../internal/cli/root_move_unix_process_test.go) and
[existing consumer source byte controls](../internal/cli/root_move_consumers_process_test.go) and
[JSON/YAML referenced collection default byte controls](../internal/cli/root_move_yaml_process_test.go).
These entry points do not establish candidate-wide, fresh JVM or cross-platform
runtime acceptance.

## Selected-owner moves

`generate|validate --config FILE --spec NAME --move-output-from SOURCE --output TARGET` moves the selected existing named owner from a v2/v4 root. The configuration still supplies the complete original single-root inventory with unchanged inputs/options; **all original inputs are read and generated** to verify exact source bytes and lock semantics before any mutation. This differs from ordinary partial generation, which skips unselected inputs. Missing/extra/renamed peers and unrelated declared roots still refuse.

The empty-target profile retains an empty ownership boundary; [compatible owned targets](output-root-merge.md) use complete target capture/CAS and merged membership. Only selected Kotlin/lock destinations must be absent; unrelated handwritten target files remain. The source keeps all peers and every shared path still owned by a peer. Shared payloads are copied to the target and remain byte-identical at source. A file owned only by the selected owner is removed at source after staging. The selected owner's raw lock payload moves unchanged; retained owners' raw locks stay unchanged. Both manifests retain v2/v4 and record their respective owner/file/lock membership; splitting intentionally rewrites manifest JSON. Kotlin and lock payload modes/mtimes remain preserved. A v4 side may retain an empty name-lock inventory without conversion to v2. Selecting the only owner uses the complete-root behavior, including exact raw manifest preservation when the target is empty.

This action reuses the entire original source observation, canonical root binding, complete sorted source/target locks and all-owned-state CAS. A retained peer's intervening code, manifest or lock change rejects apply/check/validate. Both journals stage before either applies, each manifest is last, and normal failures roll both roots back. The existing crash/recovery limits apply. Selecting the same explicit source and target is a fully validated no-op.

The report adds `moved_owners` and `retained_owners`; `files`/`name_locks` describe the selected inventory and `manifest_sha256` binds the original complete source. Full source generation remains visible in per-spec diagnostics and metrics. Check/validate/apply report the same selected membership. After success, set only the moved spec's `output_dir` to TARGET, retain peers' source roots, drop the one-shot flag and confirm ordinary full config `--check`. Keep consumer source directories consistent with the new ownership; shared declarations must not be included twice in one compilation.

The existing explicit `runtime_package` same-root requirement remains: a selection that would split that shared runtime across roots fails `E_CONFIG_RUNTIME_ROOTS` before input/state reads. Move the complete runtime-sharing inventory together. Partial move does not authorize a new runtime distribution model, incomplete source inventory, conflicting target payloads or adoption. Compatible owned targets use the separate [merge profile](output-root-merge.md).

[Selected/shared/exclusive payload and mixed-lock checks](../internal/output/root_move_owners_test.go), [sorted locks](../internal/output/root_move_owners_busy_unix_test.go), [root binding CAS](../internal/output/root_move_owners_binding_unix_test.go), [native Data/Plain × locking and post-move checks](../internal/cli/root_move_owners_process_test.go), and [native FIFO peer-state CAS](../internal/cli/root_move_owners_unix_process_test.go) define this D09/O03/T01/T02 increment. No generated Kotlin/wire or persisted-format version changes are required. General disjoint/partial adoption and other lifecycle profiles remain open.

## Source groups in disjoint configurations

A configuration may contain additional disjoint output roots when moving one explicit SOURCE→TARGET pair. The source manifest identifies the complete source owner group. Every source owner must remain in config with its original name and the same single declared SOURCE or TARGET root; missing source peers and unknown extra owners declared at moved roots still refuse. Original target owners may be declared at TARGET under the [owned-target profile](output-root-merge.md). `--spec NAME`, when present, must belong to that source group. Omitting it moves the complete source group.

Only that complete source group is generated for exact-byte/lock control and included in the report/metrics. Unrelated specs' inputs and owned output state are not read, generated, locked or changed. Their settings and paths still undergo ordinary configuration validation; declared roots must be disjoint from SOURCE/TARGET, with stable canonical ancestor bindings captured before source input reads. A binding change rejects both readonly checks and apply, including under the complete sorted source/target lock set. An unrelated owner's concurrent file or manifest update is outside this transaction.

The target may be empty or a [compatible same-version named inventory](output-root-merge.md). The latter captures and rechecks every target-owned payload before inputs and under locks. No former roots are inferred. Move one explicit pair at a time, update the moved specs' output_dir values, and confirm ordinary config `--check` with all normal inputs restored/available. Full source moves into empty targets preserve raw manifest bytes; selected-owner splits retain their manifest-membership rules above. Explicit runtime_package still requires its configured owners to share one root.

[Native full/selected × locking and post-migration checks](../internal/cli/root_move_disjoint_process_test.go), [root classification](../internal/output/root_move_unrelated_test.go), [locked canonical binding checks](../internal/output/root_move_unrelated_unix_test.go) and [native FIFO alias-rebinding checks](../internal/cli/root_move_disjoint_unix_process_test.go) define this D09 increment. Existing storage guards, whole-source CAS, staging, rollback and recovery limits remain. Kotlin/wire and persisted formats are unchanged.
