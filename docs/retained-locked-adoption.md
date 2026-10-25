# Partial locked adoption with retained original ownership

Use `generate|validate --config FILE --adopt-locked-owner --retain-locked-owner NAME`
to adopt one or more unchanged subsets of an existing locked root in one
metadata transaction. The explicit retained owner keeps the **complete original
Kotlin inventory and exact original raw lock**. Candidate specs may cover a
proper subset; nothing outside those subsets becomes unowned or is removed.

Existing [full adoption and donor addition](ownership-migration.md) can compose
this result in two commands. This action performs the transition atomically
within the existing normal-failure transaction contract. It does not promise
crash atomicity or automatic recovery.

## Selection and source authority

The root must be existing nonempty locked version3 direct output, or version4
where NAME already owns every Kotlin path in the root and has an owned lock.
On version3, the actual direct lock and manifest remain the source authority;
NAME is the explicitly created complete retained owner. On version4, existing
peers, their memberships and their locked/unlocked modes remain unchanged.
Unselected peer inputs are not read or regenerated.

Supply a nonempty candidate configuration in one root with effective hoisting,
locking and the original retained package/suffix/qualified profile. Each candidate
owner name must differ from NAME. Candidates may be new owners; an existing peer
can only replay its exact old membership and lock semantics. Changing that peer
requires the separate [reassignment contract](locked-owner-reassignment.md).
The action takes no `--spec`, donor-from, reconcile, rename, root-move or
reset/disable action. `--retain-locked-owner` requires `--adopt-locked-owner` and a
valid concrete name; it is not a TOML setting or a Boolean switch.

Before any candidate input, the command captures the complete original raw
manifest, Kotlin, locks, modes and mtimes, canonical root binding and physical
root identity. File reads use the captured canonical root; its declared binding
and physical identity are rechecked before inputs. Missing, edited, unsafe,
foreign metadata or unfinished transactions refuse. Existing storage limits and
cooperative-filesystem assumptions apply. The original full input is not required
for this action: retained code and metadata stay unchanged and are storage-verified.

Each candidate uses the existing explicit lock projection against the actual
original direct/retained raw lock. Every emitted Kotlin path and byte must match
an original owned file; every projected role/shape/name/occurrence must be an
original subset. New/changed Kotlin, allocations or occurrences refuse. Locked
singletons retain their original public classifier and descriptor. The retained
owner always covers the complete original files and allocation inventory, so
candidate subsets do not discard missing members or invent source ownership.

## Limits and metadata preservation

Original Kotlin reads are bounded at 64 MiB using the remaining actual-read
budget. Retained original Kotlin plus all generated candidates share the existing
64 MiB candidate budget; shared bytes count per candidate. Existing per-lock
1 MiB/4096-allocation/16384-occurrence limits, 16 MiB aggregate old/result raw locks,
256 resulting owner/lock limits, and manifest/parser budgets remain. Unselected
peer memberships/raw locks count in the resulting metadata budget without being
double-counted on replay. The actual chosen raw donor bytes count toward the
result limit, even when canonical candidate metadata is smaller.

On version3, the direct raw lock moves to NAME's named slot with unchanged bytes,
mode and mtime. Only the manifest-owned direct slot is removed. Version4 retains
all old raw peer payloads. New projected metadata is canonical unless an unchanged
semantic original source/donor payload can be copied; exact copies retain their
original attributes. Existing candidates retain their own raw payload on replay.
Kotlin and handwritten files are never staged, written or deleted. Manifest
version4 and lock version1 use their existing meanings.

Both readonly planning and locked apply compare the entire original raw state,
file attributes, declared binding and physical identity, including unselected
peers. Metadata stages before application and the manifest is last. Normal
failures restore original code/metadata ownership, raw bytes, modes and mtimes.
Obstructed rollback preserves recovery evidence and returns `E_RECOVERY_REQUIRED`.
The action does not retry against changed state. Configured warning policy covers
all generated candidates before planning or writes.

## Reports and subsequent generation

JSON adds `retained_locked_adoption`:

| Field | Meaning |
|---|---|
| `retained_owner` | Explicit name keeping complete original ownership |
| `before_version`, `after_version` | Actual original version3/4 and resulting version4 |
| `manifest_before_sha256`, `manifest_after_sha256` | Original/result raw manifest hashes |
| `before_owners`, `after_owners` | Sorted existing/result named inventories; version3 starts unnamed |
| `retained_files` | Complete unchanged original Kotlin path/hash inventory |
| `retained_lock` | Exact original/result raw lock hashes, profile and complete source public symbols |
| `candidates` | Candidate before/after memberships, lock hashes and projected public symbols |
| `impact` | Preservation, transaction and named-configuration guidance |

Text exposes the same retained/candidate inventories and hashes. Top-level files
and per-spec metrics describe generated candidates; retained code is not reported
as regenerated input. Absolute machine paths are omitted. Reports are point-in-time
observations, not an approval token for a later command. Exact version4 replay
preserves noncanonical raw manifest/locks and is a validated no-op.

```sh
openapi-kt generate --config new-subsets.toml \
  --adopt-locked-owner --retain-locked-owner original \
  --check --diagnostic-format json
openapi-kt generate --config new-subsets.toml \
  --adopt-locked-owner --retain-locked-owner original

# A selected ordinary update retains owners outside this candidate-only config.
openapi-kt generate --config new-subsets.toml --spec part --check
```

For a full ordinary update, include the retained NAME spec with its original full
input/options and all desired peers. **Ordinary generation without `--spec`
reconciles owners absent from its full configuration.** Do not use a candidate-only
configuration as that full inventory. Direct version3 commands refuse the resulting
version4 root. Drop the adoption flags after migration and use named configuration.

Safe changed check returns 3; validate/apply and exact replay return 0. Usage
conflicts return 2, input/profile/warning failures return 1, and storage/safety or
transaction failures return 4. Specific diagnostics are
`E_RETAINED_LOCKED_ADOPTION_INVALID`, `E_RETAINED_LOCKED_ADOPTION_CHANGED` and
`E_RETAINED_LOCKED_ADOPTION_STALE`; existing storage/profile/limit diagnostics
remain applicable. Old adoption semantics remain unchanged without the retain flag.

[Actual source/raw locks/metadata rollback/replay](../internal/output/retained_locked_adoption_test.go),
[result owner/raw-donor budgets](../internal/output/retained_locked_adoption_limits_test.go),
[complete original/peer CAS](../internal/output/retained_locked_adoption_guards_test.go),
[native locks/bindings/root identity/recovery](../internal/output/retained_locked_adoption_unix_test.go),
[bounded rollback model](../internal/output/retained_locked_adoption_fuzz_test.go),
[native profiles/singletons/policies/reports](../internal/cli/retained_locked_adoption_process_test.go)
and [native pre-input FIFO CAS](../internal/cli/retained_locked_adoption_unix_process_test.go)
define this O03/T01/T02 profile. Broader unassigned/legacy/unlocked migrations and
platform-wide acceptance remain separate.
