# Command reference

Use `--metrics` on generate/validate/check for optional CLI timing, nested core options/parse/analyze/transform/emit durations, planned file/byte counts and available Darwin/Linux process-lifetime peak RSS. Core durations sit within generate; analysis includes reference resolution/initial naming, and RSS cannot be attributed to a phase. JSON embeds metrics in the existing single report; text adds one stderr line. Default outputs/no-op/check/exit codes stay unchanged; file/stdin share read metrics and no download phase exists. Reference resolution/initial naming remain inside analyze; process RSS is cumulative. For benchmark conclusions, use repeated correctness-checked samples with equivalent inputs and host conditions.

`generate`, `validate` and `generate --check` support `--deny-warnings` with repeatable `--allow-warning CODE`. Use exact W_* identifiers, not regex/wildcards; allows require deny. Rejection is exit1/E_WARNING_DENIED before output mutation/lock acquisition (read-only name-lock load may occur first), retains original warnings and leaves generated/handwritten files untouched; defaults are unchanged. The same policy applies to file/stdin input.

The offline HTTP smoke command separately generates own-nullable raw aliases and intrinsic direct empty-object nullable roots. Public intrinsic aliases are JsonElement?; false narrowing uses actual non-null JsonElement backing, not a still-nullable alias. Reified nullable codecs can collapse JsonNull to Kotlin null and Patch.Value(JsonNull) to Patch.Null; non-null JsonElement backing still accepts JsonNull. Four Json variants cover all kinds/Patch/HTTP/fragmented SSE syntax-close-reuse-cold with independent240 fields/48 hops. Intrinsic nullable allOf/const-only/counts/shape borrowing and general graphs are not included; maximum methods compile only.

Choose the command matching the request. Paths are relative to its working directory; use explicit paths when invoking a binary from another project.

## Validate

```bash
openapi-kt validate --input ./openapi.json \
  --package-name org.example.api --diagnostic-format json
```

Validation writes no generated files. It checks the supported generation subset, not full OpenAPI/JSON Schema validity. A package is optional for validation; supply the intended package when checking its naming validity.

## File or stdin

Use `--input ./openapi.json` or `--input ./openapi.yaml` for an existing local file. Direct `generate`, `validate` and `generate --check` accept exact `--input -` (or `--input=-`) for a caller-owned stream. Stdin is read until EOF before analysis, bounded to 8 MiB plus one detection byte; excess is exit1/E_INPUT_LIMIT, read failure exit4/E_INPUT_IO. Auto stdin is JSON; YAML streams require `--input-format yaml`; empty/truncated/duplicate/trailing JSON fails safely. File input ignores stdin, help/argument errors never consume it, and `./-` is a literal file. Existing output is preserved on failure.

```bash
# Only when the caller authorizes external acquisition; not launched by the CLI.
set -o pipefail
curl -fsSL https://api.example.test/openapi.json |
  openapi-kt generate --input - --output build/generated/openapi --package-name org.example.api
```

The CLI performs no URL fetching, transport policy, retries or downloader subprocesses. Upstream exit status is the shell's responsibility: valid complete JSON followed by producer failure cannot roll back generation. For stronger acquisition guarantees, use an externally fetched local file only after producer success. Do not expose credentials or implicitly re-run a producer; each later validate/check needs a fresh stream/file. A pipe is not incremental generation and may block while awaiting EOF.

Breaking source migration: `download`/`help download` are removed (usage exit2/E_USAGE), `[specs.download]` and URL/header metadata are rejected, not ignored. Keep local configs but remove download tables and fetch externally; configured `input = '-'` is invalid (E_CONFIG_INVALID), `./-` is an ordinary file. No config stdin replay exists. Output/config/name-lock versions and generated Kotlin APIs/wire are unchanged. Historical alpha binaries may still have the downloader; confirm the selected executable supports stdin via actual help before piping.

## YAML profile

JSON and YAML share the ordered tree, analyzer, warnings and emitter. Equivalent accepted trees with the same declaration/list order and number tokens yield identical Kotlin/API/wire/semantic Pointers. Auto recognizes `.yaml`/`.yml` case-insensitively; stdin/extensionless auto is JSON. Explicit YAML also accepts JSON syntax under its scalar/Unicode rules; config input_format='yaml' works for local files.

Scalar string mapping keys and unique decoded keys/anchors are required. Defined finite value aliases expand in occurrence order. Non-string/complex/alias keys, duplicate keys/anchor rebinding, cycles/undefined aliases, merge keys, custom/non-JSON tags, explicit timestamps, binary/sets/pairs and multi-documents are rejected. Implicit dates stay strings; quote date keys and the string null in 3.1 type lists. True/false and null/~ /empty values become JSON scalars; yes/no/on/off stay strings. Only finite JSON number syntax is accepted, including on resolver overflow; quotes/!!str preserve explicit strings. Builtin map/seq/str/int/float/bool/null tags require compatible kinds, and !!int requires integer syntax. Custom URI identities such as !<!!str>, !<%21%21map> or !%21seq fail even when Node.Tag normalizes to a builtin spelling; standard !! shorthand and canonical tag:yaml.org,2002: URIs remain accepted with valid percent encoding and unchanged kind/value checks. %TAG source lines are rejected before parsing to prevent URI-prefix allocation amplification, including BOM-prefixed/quoted-continuation lines and all YAML line break forms; indent string data or escape percent.

Raw and expanded compact JSON are each at most 8 MiB, source/expanded traversal each 100,000 visits including keys/aliases, containers and alias paths at most 127. Source tree checks follow the upstream bounded Node parse (syntax nesting limit 10,000); this is not a hard RSS guarantee. UTF-8 is required, escaped surrogate/out-of-range Unicode fails. Existing schema/JVM caps and output ownership stay intact. E_YAML_INVALID / E_YAML_UNSUPPORTED / E_YAML_ALIAS_CYCLE / E_YAML_LIMIT are payload-safe exit1 diagnostics, with available node coordinates. Full YAML/general source spans are outside this profile; historical binaries may still reject YAML.

## Generate

```bash
openapi-kt generate --input ./openapi.json \
  --output build/generated/openapi --package-name org.example.api \
  --model-name-suffix Dto
```

Generation validates before output mutation. Review emitted files and approximation warnings, then reuse the generation options with `--check` to confirm consistency. For HTTP naming/filtering:

```bash
openapi-kt generate --input ./openapi.json \
  --output build/generated/openapi --package-name org.example.api \
  --api-client-name CatalogClient \
  --exclude-api InternalApi --exclude-api AdminApi \
  --diagnostic-format json
```

All operations are validated before exclusion; filtering cannot suppress unsupported semantics. HTTP naming/filtering options do not change model-only artifacts.

## Check drift

```bash
openapi-kt generate --input ./openapi.json \
  --output build/generated/openapi --package-name org.example.api \
  --model-name-suffix Dto --check
```

Match input, output, package, suffix, client name and exclusions to generation. Preserve the exit status in CI. `--check` compares contents, stale owned files and manifest state without creating or modifying output or acquiring a writer lock. It is a point-in-time comparison, not isolation from concurrent edits. A check-only request ends with the result; updating drift is a separate requested action.

## Resolved ordinary model merging

Use `--merge-identical-models[=true|false]`, core `MergeIdenticalModels`, or config `merge_identical_models` (default false). Eligible ordinary DTO/String-enum owned roots compare acyclic resolved typed references, preferring component then shared then API lexical owners. Different checks/defaults/codec/Patch/order stay separate. Repeat `--exclude-merge-model FQ_NAME` / core `ExcludeMergeModels` / config `exclude_merge_models` for exact fully qualified generated root classifiers, including suffix; exclusions require merging and neither absorb nor are absorbed. No regex in this subset. CLI lists/false and config empty lists replace inherited settings; complete inventory is validated before selected input/state reads.

Pattern/suffix exclusions use config `exclude_merge_model_patterns` / `exclude_merge_model_suffixes` or repeatable CLI flags. Each list replaces only its corresponding effective list; spec `[]` resets it. Patterns match the whole generated FQ root, suffixes the literal short name including the configured model suffix, case-sensitive. Maximum 64 patterns (1,024 bytes/256 syntax nodes, finite repetition bounds ≤64) and 64 suffixes (1–256 ASCII letters/digits/underscores). Lookaround/backreferences/flags/non-ASCII/shorthand/nested classes/lazy/possessive forms fail before input reads. They affect merge only; prior hoist/dedup is separate.

Old root names remain typealiases: constructors/data copy/canonical companions and reified codecs use the canonical owner, but class/descriptor/equality/plain toString identity changes and nested/singleton accesses must migrate to canonical owners. Rebuild compiled consumers. Opaque provider/code owners remain protected; cyclic/unresolved/over48comparison graphs skip merging without new input errors. Pass order hoist/locked-hoist→dedup→merge→plain; lock names/fingerprints are preserved before merging, not reset. Newly collapsed nominal overloads use existing path/method numeric suffix allocation after merge, reserving natural names (second submit→submit3 if submit2 exists); migrate affected method calls, not operation paths or old model aliases. Broader union/hierarchy/general recursive graphs/regex/all-combinations/platform/performance remain unverified.

## Hoisted-name state

Use `--hoist-shared-shapes --lock-hoisted-names[=true|false]` or config boolean
`lock_hoisted_names` for the initial repeated ordinary operation DTO/enum roots.
Default false; bare flag true; use exact inline booleans. Reuse the same input,
package, suffix and qualification profile for generate/check/validate. Direct
locked validation needs `--output` as a read-only state root and permits
`--model-name-suffix`; those validate arguments require effective locking or an
explicit disable action.

Keep `.openapi-kt-names.json` and version3 manifest for direct output; config
version4 owns separate `.openapi-kt-names-<owner SHA256>.json` payloads. Reports
list only Kotlin files. Config false replaces inherited true only for fresh
owners: existing selected lock state cannot implicitly be disabled. Unselected
owners retain state; full reconciliation removes only manifest-owned state.
Version4 is not silently downgraded. Exact-match version1 adoption is available;
version3-to-version4 uses [locked-owner adoption](#adopt-a-direct-locked-owner).
[Explicit complete root move](#move-an-output-root)
preserves existing versions and raw state. [Configured owner
rename](#rename-a-configured-owner) preserves existing version2/4 Kotlin and lock
semantics through a separate one-shot mapping.

For an authorized disable/reset, keep hoisting enabled and the same stored
package/suffix/qualification profile. Pass the one-shot
`--name-lock-action reset|disable` to generate/validate, also with `--config FILE
[--spec NAME]`. Reset needs effective locking true and regenerates fresh state;
disable needs locking false and regenerates fresh names before removing owned
state. Both require existing valid state; they do not repair corruption. Store
only the final boolean mode in configuration; there is no TOML action key.
Full selection applies the action to every selected spec, while `--spec` retains
unselected owners, locks and shared runtime membership. Combine neither action
with `--adopt-legacy`, `--adopt-locked-owner`, `--move-output-from`, `--rename-owner` nor empty `--reconcile`.

```bash
openapi-kt generate --input ./openapi.json --output build/generated/openapi \
  --package-name org.example.api --model-name-suffix Dto \
  --hoist-shared-shapes --lock-hoisted-names \
  --name-lock-action reset --check --diagnostic-format json
```

Review `name_lock_migrations`: complete old/new hoisted classifier and canonical
descriptor inventories, exact role+Pointer changes, relative Kotlin paths with
old/new SHA256 for selected owner membership (shared paths may remain retained
by peers), and migration guidance. Earlier representatives can rename
shared imports and nested classifier owners; API aliases retarget, descriptors
and JVM class identity change, and compiled consumers need rebuilding. Shape
changes additionally need constructor/validation/wire review; this report is not
an API/wire equivalence proof. Apply the same requested action without `--check`
only within the user's generation authorization. Action validate/check are
read-only including output preflight; check differences exit3, safe validation
exit0. Plain locked validation keeps its retained-state compatibility semantics.
Direct disable explicitly changes v3 to v1; named output remains v4 even after
its final lock is removed. Keep locking false for subsequent config generation.
A completed disable cannot be repeated on absent state; ordinary generation is
the next step. Normal failures restore code, manifest and metadata together.
Read-only previews are point-in-time comparisons; each apply rereads and compares
exact old bytes under the cooperative lock set.

`E_NAME_LOCK_STALE` means a cooperative writer changed state after loading:
retain its files and rerun the requested command with current input/state.
Missing/edited/malformed/profile/shape/removal errors require preserving and
reviewing old source/state, not deleting metadata to force success. Check and
validate write nothing; validate checks core state compatibility, not a concurrent
CAS snapshot. There is no automatic repair or hostile-filesystem/crash guarantee.

## Rename a configured owner

Use invocation-only, repeatable `--rename-owner OLD:NEW` (or
`--rename-owner=OLD:NEW`) with `generate|validate --config FILE`. Change the
configured spec name to `NEW`, retaining the existing input, output root and
emission options. `OLD` must be an existing version2/4 owner in that selected
spec's same effective output root. Names are case-sensitive ASCII identifiers
starting with a letter, followed by letters, digits, `_` or `-`. Each old/new
name is unique across the invocation. There is no TOML rename key; combine
neither `--adopt-legacy`, `--adopt-locked-owner`, `--move-output-from`, `--reconcile` nor `--name-lock-action` with rename.

With `--spec NEW`, provide exactly one mapping targeting that current spec.
Without `--spec`, map every current configured spec; use `events:events` for an
unchanged owner. An occupied new owner must also be an old source in the
complete selected mapping for that same root; this permits simultaneous swaps
and chains while retaining every source's exact Kotlin/lock identity. Keep each
new spec's input/options consistent with its mapped old owner. Identity and a
permutation producing identical memberships/raw payloads are validated no-ops.
Unknown sources and unselected occupied targets fail closed. The command reads only
selected config roots, with no former-root discovery, root move, version1
adoption or direct version3-to-named-version4 transition. Partial selection
leaves unselected inputs and other roots unread.

```bash
# The current configured spec is inventory; its existing owner is catalog.
openapi-kt generate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory --check --diagnostic-format json
openapi-kt validate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory --diagnostic-format json
# Apply within the user's owner-rename authorization.
openapi-kt generate --config openapi-kt.toml --spec inventory \
  --rename-owner catalog:inventory
# Confirm subsequent generation under the current owner.
openapi-kt generate --config openapi-kt.toml --spec inventory --check
```

Review additive `owner_renames`: exact `old_owner`/`new_owner`, manifest `version`,
sorted unchanged `files[{path,sha256}]`, optional
`name_lock{sha256,profile{package,suffix,qualified},symbols}` and `impact`.
Each symbol records `role`, `shape`, public `classifier`, canonical emitted
`descriptor` and original `occurrences`. Text exposes the same data. The
top-level `files` list remains Kotlin-only. Reports are point-in-time
comparisons, not snapshots or approval tokens binding a later apply.

The selected candidate must retain exact Kotlin membership/bytes, locked or
unlocked mode and locked profile/allocations/occurrences. Rename reads and
validates the manifest and all owned state/files in each selected root,
including peers absent from current config; missing, edited or malformed
metadata or Kotlin is rejected. Unowned new metadata destinations collide even
if identical. Unselected peer owners and their metadata remain intact; shared
memberships follow their explicitly mapped owner names. Full rename performs
no reconciliation.

Preflight and apply under the complete sorted cooperative root lock set compare
exact manifest and every owned lock's bytes. Stale bytes are rejected without
automatic regeneration/retry. Apply stages metadata only, with each manifest
last; a locked owner's exact old payload moves to the filename derived from
SHA256 of `NEW`. Overwritten destinations are staged once, and old filenames
are removed only when they have no incoming locked owner. Kotlin is never written or removed. Normal failures roll back
metadata across roots; failed rollback retains journals/backups and returns
`E_RECOVERY_REQUIRED`. Busy apply returns `E_OUTPUT_BUSY`; recovery evidence
refuses read/check/validate/apply. No crash atomicity or automatic recover is
provided.

Rename check/validate perform read-only preflight without locks, directories or
stages: check exits 3 for a safe change, 0 for a validated no-op; safe
validation exits 0. Read-only commands do not exclude busy writers. Core profile
errors remain exit1; parser/flag errors are exit2; storage/rename refusals are
exit4 (`E_OWNER_RENAME_INVALID`, `E_OWNER_RENAME_CHANGED`,
`E_OWNER_RENAME_STALE` or existing safety diagnostics). Preserve state on
refusal. After success, drop the one-shot old mapping and keep the new name for
ordinary generation. An old mapping is a new request against current ownership,
so missing owners or mismatched current candidates fail rather than replaying
the previous transaction.

## Move an output root

Use one-shot `--move-output-from SOURCE --output TARGET` with generate or validate.
Both CLI roots resolve from the invocation directory. Bind the original input and
all emission options; direct accepts existing v1/v3, configuration existing v2/v4.
The source group must be full, nonempty, same declared root and exactly all original owners
with their original names/inputs/locked modes, including retained/shared peers.
The declared config root resolves from the config directory and must be nonempty
after defaults/spec overlay, before the CLI target override; missing/empty roots
(including empty spec overrides) fail exit4/E_OUTPUT_ROOT_MOVE_INVALID before
owned-state or input reads. TOML empty-value and ordinary override semantics stay
unchanged. The declared root must canonically
match source or target. For this action only, full inventory permits the explicit
output override. Extra/missing source owners,
`--rename-owner`, `--adopt-legacy`, `--adopt-locked-owner`, reconcile and lock lifecycle actions are rejected.
There is no TOML action key, root discovery or version adoption/conversion.

```bash
openapi-kt generate --input ./openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new \
  --check --diagnostic-format json
openapi-kt validate --input ./openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new --diagnostic-format json
# Apply within the user's explicit root-move authorization.
openapi-kt generate --input ./openapi.json --package-name org.example.api \
  --move-output-from generated-old --output generated-new
openapi-kt generate --input ./openapi.json --package-name org.example.api \
  --output generated-new --check
```

Review `output_root_migration`: preserved manifest version/raw SHA256, identity,
sorted `files[{before,after,sha256,owners}]`, owned `name_locks` with relative
path/raw SHA256/profile and complete hoist classifier/descriptor/occurrence
inventory, plus impact guidance. Paths are relative to the selected source/target
and remain equal; absolute roots are omitted. Exact Kotlin/manifest/lock bytes,
file modes/mtimes, owner names/shared memberships and lock semantics are retained.
Update the consumer source directory and final configured root; drop the one-shot
flag after success. Replaying an emptied source fails. Same explicit root is a
validated no-op; distinct aliases and overlaps refuse planning.

All owned files/state must be regular, present, hash-matching and valid. Foreign
`.openapi-kt-names*` metadata and unfinished stages refuse migration. Target must
have no manifest/name metadata or occupied moved payload path, even if identical;
other unowned files are preserved on both sides. Source cleanup deletes only
verified owned payloads/manifest, preserving its root, directories and sidecar.
Normal root ancestor-resolution and internal symlink/path guards apply. Changed
Kotlin or lock mode/profile/allocations/occurrences refuse movement; change the
spec separately. No Kotlin code or state format is rewritten by migration.

Check/validate are read-only preflights without locks or directory creation;
check differences exit3, identity check/safe validation exit0. Apply holds the
complete sorted source/target locks, rechecks initial canonical source/target/config
bindings and exact
code/manifest/all-lock bytes and attributes plus target absence, stages both roots
before any apply and journals each manifest last. Normal failures restore both
roots; obstructed rollback preserves journal/backups and returns
`E_RECOVERY_REQUIRED`. Storage/migration failures exit4, including
`E_OUTPUT_ROOT_MOVE_INVALID/CHANGED/OCCUPIED/STALE`; preserve output on refusal.
Existing input/core/warning failures retain their exit policies. There is no
cross-root crash atomicity or automatic recovery, and read-only reports do not
bind a later apply or exclude busy writers. Partial/disjoint inventory moves
remain unsupported. Root move preserves the manifest version; use the separate
[locked-owner adoption](#adopt-a-direct-locked-owner) workflow for same-root v3→v4.

## Adopt a direct locked owner

Use the unique invocation-only bare `--adopt-locked-owner` on
`generate|validate --config FILE [--spec NAME]`. With explicit `--spec`, the
configuration must contain exactly that one spec, claiming the complete old
Kotlin inventory and raw lock. Without `--spec`, every configured spec must use
the same existing locked root; different file subsets and shared memberships
are allowed only when their union exactly preserves all original Kotlin paths
and bytes. All specs require hoisting/locking and the original effective profile.
No TOML action key; fresh/unlocked/incompletely covered roots and combinations
with legacy adoption, rename, root move, reset/disable or reconcile refuse. Existing `--adopt-legacy` version1 behavior remains unchanged. Explicit
`--output` can fill an omitted/empty declaration for this existing same-root
adoption; root move still requires a nonempty declared root before its target
override and before state/input IO.

```bash
openapi-kt generate --config openapi-kt.toml --spec catalog \
  --adopt-locked-owner --check --diagnostic-format json
openapi-kt validate --config openapi-kt.toml --spec catalog \
  --adopt-locked-owner --diagnostic-format json
# Apply within the user's explicit ownership-migration authorization.
openapi-kt generate --config openapi-kt.toml --spec catalog --adopt-locked-owner
openapi-kt generate --config openapi-kt.toml --spec catalog --check
```

Review `locked_owner_adoptions`: owner, old/new manifest versions/SHA256,
complete unchanged Kotlin path/hash membership, raw lock before/after SHA256,
profile and symbols with public classifier/canonical descriptor/occurrences,
and impact. Pre-input capture binds original canonical root identity and exact
manifest/all Kotlin/raw lock bytes; check/validate and apply repeat CAS, with
apply under sorted locks. This action adds a 64MiB aggregate old-Kotlin capture
limit, separate from the existing per-file bound. Check every file size and the
complete sum before any Kotlin payload capture; rereads also check remaining
budget and hashes. Overflow is E_OUTPUT_LIMIT before content hashing. Existing
manifest/lock/core/config budgets remain; this is no hard RSS/benchmark claim. Missing,
edited, occupied, symlink, alias rebound, stale or recovery state refuses.
Read-only reports are point-in-time comparisons, not later-apply approval tokens.

Check exits3 for safe v3→v4 metadata change; safe validate exits0. Both perform
complete read-only preflight with no dirs/sidecars/stages/mtime writes. Apply
moves the exact owned anonymous raw lock to its owner-hash filename and commits
v4 full membership/lock mapping last. Kotlin bytes/modes/mtimes, handwritten
files, foreign metadata and other roots stay intact. Normal rollback restores
old manifest/anonymous lock attributes; obstructed rollback attempts remaining
restoration and retains journal/backups with E_RECOVERY_REQUIRED. No crash
atomicity or automatic recover. Keep metadata on refusal.

After success, ordinary management must use config with this named owner;
direct v3 refuses v4. Exact same-owner complete v4 action replay is a validated
no-op, including valid noncanonical raw state. Later intentional updates belong
to ordinary named generation, whose canonical lock-writing behavior remains.
Input-byte history is not separately persisted: acceptance binds generated and
persisted identities. This mode does not move roots or reconcile. Existing-v4
owner additions use the explicit donor mode below.

For a full-config split, omit `--spec` in the commands above. Each projected lock
retains original role/shape/name and a subset of original occurrences; their union
must exactly equal the old lock. Even a singleton keeps its old canonical hoist,
which ordinary named generation can replay. Projection canonically encodes a
changed subset; unchanged semantics reuse the exact original raw payload. Review
each owner's files/symbols/raw-lock hashes and the first proposal entry's complete
`source_symbols`, alongside exact manifest hashes. Older generators do not
implement singleton retention. Candidate Kotlin has a separate 64MiB aggregate
limit, counting shared bytes per candidate; owners256, named locks16MiB and
conservative ownership metadata64MiB remain bounded. The transaction writes only
named lock payloads, anonymous-lock removal and manifest last. Full v4 replay
requires exact owner/lock inventory and each membership, preserving raw metadata;
it cannot redistribute existing memberships. Keep incomplete adoption separate
from complete split coverage.

## Inputs and options

Support is one local OpenAPI 3.0.x / 3.1.x JSON or bounded YAML document. Model-only inputs need nonempty `components.schemas`; supported HTTP inputs may omit components. YAML follows the profile above; external references are not implemented; specification acquisition is outside the product, not a pending download feature. A bounded local TOML `generate|validate --config FILE [--spec NAME]` workflow is available for disjoint output roots with normal-failure batch rollback: config_version=1, [defaults]/[[specs]], name/input/output_dir/package_name/model_name_suffix/api_client_name/runtime_package/model_style/merge_identical_models/exclude_merge_models/deduplicate_models/hoist_shared_shapes/qualify_hoisted_names/lock_hoisted_names/exclude_apis/validation/input_format; bounded TOML 1.1 syntax for the closed schema: quoted/dotted keys, basic/literal/multiline strings, inline tables, arrays/array-of-tables and standard integer spellings (config_version integer value 1); fields still require their declared string/boolean/string-array/table types, so parsed dates/floats and nested/mixed arrays in those fields fail explicitly. Download tables/metadata and plugin DSL are rejected. Limits are 1 MiB, 256 specs, 64 structural levels, 65,536 AST nodes and 33,554,432 node/key-comparison work units; errors report only known schema locations/closest containers without arbitrary parser excerpts or unknown keys. Config inputs are local paths, not '-' stdin markers (use './-' for a literal file). Config paths resolve from the config directory; explicit CLI overrides require one selected spec. It uses version2 ownership by default or version4 with per-owner name-lock metadata, keeps unselected owners, reconciles removed owners only on ordinary full generation, and requires explicit --reconcile/default output_dir for an empty inventory. Check writes nothing; failed generation preserves old output. Version1 is not adopted automatically. For an authorized switch from a direct-input tree, preserve the original document/options, use generate --config FILE --spec NAME --adopt-legacy --check (3 for metadata change), then --adopt-legacy without --check only after confirming that spec owns the entire legacy file set. First adoption requires exact paths/hashes and existing regular unmodified owned files, changes only manifest metadata, and refuses mismatches/missing files; do not use it to rename/move/split owners or reset a corrupt manifest. On version2 it uses normal named updates, idempotent for unchanged inputs. RuntimePackage/--runtime-package/config runtime_package selects existing Patch/bigint utilities (empty uses historical <package>.models), not all API-private helpers or standalone runtime libraries; utility bodies/visibility and JSON codecs are unchanged. Explicit shared package declarations must resolve to one declared output-root path across all configured specs, including unselected/matching-default peers; shared identical files retain all owners and last-owner cleanup. A runtime package change intentionally changes Patch imports and bigint annotations; compile the sources in one Kotlin module because bigint serializer is internal. Configuration --metrics reports separate config_bytes and accumulated selected-input read/generate timings; planned files/bytes deduplicate identical(root,path) candidates only after every selected core generation succeeds, and are zero for incomplete/conflicting envelopes. Planned counts are not writes or retained-owner counts; errors/deny/check/noop preserve output. Fetch files externally before local configuration generation; the CLI never launches curl or contacts spec URLs. Direct stdin uses the same supported subset and fails safely before output, but each later command needs a new stream/file. Cross-root crash atomicity/automatic recovery remain unavailable; exact same-root complete or full-config split version3-to-version4 adoption uses its separate workflow above; complete/selected-owner/disjoint-source-group same-version root move uses the explicit workflow above; exact same-root version2/4 owner rename uses the separate workflow above; explicit lock disable/reset has the bounded workflow above; metrics do not provide independent core stage timing, RSS or speedup claims. Preserve existing output and report the diagnostic rather than changing the manifest version. A formal Gradle generator plugin is not planned; use the local CLI and configure the consumer project's generated source directory. --model-style/config model_style supports empty/data_class(default) or plain_class: identical wire/constructors/getters/serializer checks, intentional removal of data-class copy/componentN and toString becomes class-name-only. Explicit field equality/hash retains NaN, signed-zero, nullable/collection and array data semantics (arrays reference equality but static-array content hash), provider typealiases resolved by Kotlin. Known singleton/data objects, zero-field const class identity, aliases/valueclasses/provider classes and sharedPatch.Value stay unchanged. The independent initial --deduplicate-models[=true|false]/config deduplicate_models/core DeduplicateModels option folds only equivalent operation ordinary DTO/String-enum owned trees, keeps old root names as aliases to lexical canonical classes, and does not merge different checks/defaults/codecs/Patch/nullability/property order/external nominal types. Root constructors/canonical companions remain, but alias nested classifier accesses (including sealed-enum known data objects such as FAST) must migrate to canonical owners or use retained fromSerialName/entries companion APIs; JVM class/serializer descriptor identity and cross-old-name equality become canonical. Explicit false resets inherited true; multi-selected CLI overrides are rejected. Initial --hoist-shared-shapes[=true|false]/config hoist_shared_shapes/core HoistSharedShapes independently relocates repeated same-recorded-role ordinary DTO/String-enum owned roots to <package>.models; all former API root names become aliases. Recorded role+suffix supplies shared short names (ResponseDto), --qualify-hoisted-names/config qualify_hoisted_names/core QualifyHoistedNames keeps canonical owner-qualified names (AlphaResponseDto) and requires effective hoist true. Casefold component/helper names reserve base/base2/base3; not name-lock stability under adding collisions. Both passes protect owners mentioned by opaque incoming provider/type/code expressions instead of rewriting them. Migrate even former-canonical API-root nested accesses to shared models owners (including sealed-enum known data objects such as FAST; retained fromSerialName/entries companion APIs can retrieve them), class/descriptor package identity changes; old root constructors/companions/Data copy remain and Plain no-copy remains. Reset both hoist and qualification to false together when inherited qualified true. Broad nested-only/union/hierarchy/SSE/dictionary/error-shape hoisting/general merge remain outside this slice; initial persisted names are covered under Hoisted-name state above. Neither option demonstrates compiler/RSS/speed optimization; rebuild callers rather than assuming source/binary API compatibility. There is no direct-URL/download command; SSE has a bounded typed cold-Flow subset; use the Ktor guidance for plugin, cursor and streaming-test requirements. HTTP and schema support depend on context; consult diagnostics and matching contracts for unsupported constructs.

Pure root references to supported non-null ordinary object DTOs emit native typealiases; consumers share the target's canonical constructor, serializer and nominal identity. Alias/direct HTTP collisions receive ordinary suffixes. Every alias hop counts toward depth. Pure root collection bindings also support native List/typed Map/JsonObject targets, preserving canonical checks/peers and every source hop. Root alias true/false/absent intent and consumer overrides are distinct from item/value nullability; false narrows new root-bound nullable maps without altering historical direct dictionary behavior. Accepted non-null enum/valueclass/native/const/special scalar roots also admit pure bindings, retaining canonical constructor/serializer/defaults/wire and actual slot cost. Nested enum classifiers stay on the canonical owner; builtin-only scalar root aliases remain validation-opt-in. Non-discriminator standalone union root bindings preserve canonical serializers and variant owners, including Unknown/native-kind/null/closed wire. Construct nested variants on the canonical root, not the alias. Hierarchy parent/leaf bindings also retain canonical graph/mapping/tag owners; use original leaf constructors and serializer APIs, not abstract parent constructors. Alias/direct duplicate branches are diagnosed and unmapped fallback wire uses the canonical component key. Raw empty-object/empty-allOf root bindings remain JsonElement passthrough: use JsonElement.serializer(), not an alias companion or DTO constructor. Previously conflicting raw overloads get ordinary numeric suffixes. Own-nullable ordinary DTO aliases also retain nearest true/false intent and request Patch/required-nullable behavior while the terminal DTO stays non-null. `NullableAlias.serializer()` still resolves the non-null canonical companion: use reified `serializer<NullableAlias>()` or `Canonical.serializer().nullable` for null values. Intrinsic nullable direct DTO components with constructor fields also retain the same physical classifier and inherit typed `T?` unless explicitly narrowed false. Use the canonical nullable adapter/reified nullable type for root null; raw/const-only/nullable-allOf/whole-count and shape-borrowing contexts remain rejected. Own-nullable plain typed/typeless String-enum aliases retain canonical entries/fromSerialName/open future values and nearest root overrides. Nullable alias companions remain non-null; use reified/nullable-adapter serializers for null and canonical owners for nested classifiers. Own-nullable primitive value-class aliases preserve canonical constructors/bounds, primitive wire and boxed versus narrowed Long/Double slots; intrinsic nullable primitive wrappers independently preserve physical classifiers and typed inherited/false-narrowed intent, while native/date/byte/const value-class wrappers remain unsupported. Own-nullable temporal aliases independently preserve non-null date/date-time/date-native terminals and canonical Instant/LocalDate parse/nullable codecs/ISO wire; intrinsic temporal components independently export transparent nullable native root aliases with reified nullable codecs/false native backing; own-nullable bigint aliases independently retain Long extrema APIs/nullable reified codecs and direct-field/Patch decimal strings versus root/list/map JSON numbers; intrinsic nullable bigint components separately export Long? aliases with nullable reified root codecs and inherited/false backing while preserving the same field-string/root-number decision. Own-nullable byte/binary aliases preserve builtin signed-array JSON model/Patch/SSE codecs with nearest true/false intent, but simple Ktor HTTP roots remain raw: null requests send ASCII `null`, and that response is a non-null byte array, not Kotlin null. Do not assume Base64 or HTTP JSON-null round-trips. Intrinsic nullable byte/binary components separately export ByteArray? roots and inherited/false backing with the same JSON-array versus raw-HTTP projection. There is no alias `.serializer()` companion: use reified aliases or `ByteArraySerializer().nullable` for JSON null. Own-nullable caller-native pure aliases separately retain accepted non-null provider classifiers/codecs with nearest true/false intent. Constructor/companion APIs depend on the provided class; no replacement codec is generated. Native Long fields remain numbers (not bigint strings), while native String/ByteArray HTTP roots remain raw and `null` responses are text/bytes, not Kotlin null. Intrinsic nullable caller-native components separately export transparent provided.Type? aliases with reified nullable root codecs and direct/wrapper/alias inherited/false intent, without copying provider classifiers/serializers or changing wire. Shape borrowing remains rejected even through false aliases or unused mappings. Temporal/native enums/const and structured custom-native schemas remain excluded. Intrinsic nullable plain String enums independently retain the physical vocabulary and typed inherited/false-narrowed root intent; null enum members and formatted/native enum nullability remain unsupported. Intrinsic nullable ordinary String/Boolean/Int/Long/Float/Double roots independently export transparent builtin T? aliases; references/wrappers and pure aliases to those intrinsic targets inherit nearest intent or narrow to actual builtin backing. Use reified serializers or builtin nullable adapters, not an invented generated companion. Typed bound checks still require `--validation`; bare alias assignment/standalone primitive serde is unchecked. Root String HTTP is raw text: the response text `null` is not Kotlin null. Own-nullable aliases to accepted non-null ordinary builtin primitive targets separately expose T? with inherited/false backing; constraints still require `--validation`. Unmarked/false-only aliases directly to non-null targets remain validation-opt-in; an unsupported intermediate is not bypassed by a nullable caller. Canonical bound/source identity survives accepted legacy intermediate aliases, and alias assignment/standalone serde stays unchecked. Direct property-owned nested DTO schema Pointers separately bind existing canonical `models.OwnerDto.Node` classifiers, with nominal recursion and consumer nullable/Patch behavior. Finite internal ref chains to those DTOs separately preserve nearest nullable intent and full erased-hop/cache-height budgets; transparent cycles and nominal leaves in borrowed builtin containers remain excluded. Direct List/typed Map-owned DTO targets additionally retain physical nested Inner/Value classifiers and root top-level peers (RowsDtoInner, not RowsDto.Inner), without inheriting container nullable intent. Annotation-only component-root aliases to eligible internal DTOs retain exported typealiases and the original physical constructor/serializer. Finite typed internal chains may end at direct named ordinary DTOs or existing named aliases to eligible direct/internal DTOs; nearest internal nullable intent precedes effective alias intent, and the consumer overrides both. Component-root aliases may also cross those internal chains into eligible named DTOs/DTO aliases, reusing the physical constructor/serializer. Own root intent precedes internal/named intent; explicit consumer true/false overrides all of them. Forward dependencies and transparent cycles retain the catalog DFS policy. Every internal/named alias hop remains in source depth after narrowing and transparent collection caching. Nonordinary named terminals, composition/union/hierarchy owners, fixed-object extras scratch and wider internal nominal graphs remain unsupported; target bounds still require `--validation`.

Direct array components may be nullable. Absent/true reference overrides retain the nullable alias; explicit false uses a genuinely non-null backing collection, including qualified item peers. Shared model defaults and inline request required/Patch presence stay context-specific; standalone aliases do not enforce typed-consumer constraints. This does not add nullable object/primitive roots or root-ref bindings.

Output arguments containing `..` are rejected. Packages must use valid ASCII Kotlin segments, without keywords, reserved namespaces or path traversal.

| Flag | Commands | Contract |
|---|---|---|
| `--input` | Both | Required single local file |
| `--package-name` | Both | Required for generate; validate defaults to a neutral internal package |
| `--input-format auto\|json\|yaml` | Both | Auto selects `.json`/`.yaml`/`.yml`; extensionless/stdin is JSON. Explicit YAML follows the bounded profile; JSON errors never fall back to YAML |
| `--diagnostic-format text\|json` | Both | Default text; structured result with JSON |
| `--api-client-name CatalogClient` | Both | Override only the HTTP aggregate name |
| `--exclude-api WidgetsApi` | Both | Repeatable, exact case-sensitive normalized API class name; absent names are ignored |
| `--output` | generate; locked/action/config rename/root move validate | Output root; required in direct state-aware validation |
| `--model-name-suffix Dto` | generate; locked/action/config rename/root move validate | Suffix component/operation-local names; use the matching lock profile |
| `--merge-identical-models[=true\|false]` | both | Default false; initial resolved ordinary roots, canonical aliases |
| `--exclude-merge-model FQ_NAME` | both | Repeatable exact root classifier, requires merge, not regex |
| `--exclude-merge-model-pattern REGEX` | both | Repeatable whole-FQ-root portable ASCII pattern, requires merge |
| `--exclude-merge-model-suffix SUFFIX` | both | Repeatable case-sensitive literal short-root suffix including model suffix, requires merge |
| `--lock-hoisted-names[=true\|false]` | both | Default false; requires hoist; reuse transactionally stored metadata, not a reset flag |
| `--name-lock-action reset\|disable` | both | One-shot existing-state lifecycle; matching profile/hoist, target locking true/false, migration report and transactional CAS |
| `--rename-owner OLD:NEW` | both, with `--config` | Repeatable one-shot same-root v2/v4 transfer; map every selected spec, preserve exact Kotlin/lock semantics and all peers without reconciliation |
| `--adopt-locked-owner` | both, with `--config` | Explicit one-spec complete adoption, or no-spec full-config split with exact Kotlin/allocation-occurrence union; metadata-only same-root v3→v4, exact v4 replay; no incomplete coverage |
| `--adopt-locked-owner-from OWNER` | both, with adoption/config/explicit different `--spec` | Existing-v4 donor subset addition; preserves all old peer memberships/raw locks and exact Kotlin; no TOML key |
| `--retain-locked-owner NAME` | both, with --adopt-locked-owner/full candidate config | Retain complete original v3/v4 locked ownership under NAME while adopting exact subsets; no --spec/donor/action combinations |
| `--reassign-locked-owners` | both, full config only | Atomic same-root fully locked v4 desired-owner redistribution; preserve exact Kotlin/historical allocation union; no spec/adoption/reconcile/rename/move/reset |
| `--move-output-from SOURCE` | both, with explicit `--output TARGET` | One-shot complete same-version v1–v4 root move; direct or full original same-root config inventory, exact code/lock semantics, optional config --spec selection and compatible named owned target, no adoption/rename/action |
| `--check` | generate | Read-only comparison |
| `--validation` | both | Bare opt-in DTO/ref/element/container bounds flag; default off. Use consistently in generate/validate/check; no valued or repeated form |

Use the selected binary's `--help` for the executable flag inventory.

## Diagnostics and output safety

Text warnings go to stderr. JSON mode emits one document on stdout: `diagnostics` contains errors, and `warnings` appears separately when nonempty. Warnings change neither exit codes nor file content. Review `W_` diagnostics as approximation notices, not promises of runtime schema enforcement.

Diagnostics carry code, message and optional original-input JSON Pointer. An absent pointer differs from an empty string (document root). Report safe messages and locations without exposing private input payloads.

| Exit | Meaning | Action |
|---|---|---|
| `0` | Success | Review warnings |
| `1` | Invalid/unsupported input or generation error | Use code/pointer to locate the context and propose a fix, preserving the original spec |
| `2` | Usage error | Check subcommand help |
| `3` | Safe check differences | Report drift; regenerate only when an update is requested |
| `4` | Input IO, filesystem or output-safety failure | Inspect the selected stream/file, ownership, edits, paths, locking or recovery state before changing files; retain old output |

`.openapi-kt-manifest.json` records version and owned paths' SHA-256 hashes. Keep it with corresponding generated files, including when checking them into version control. Updates remove only stale owned files and preserve unowned files. Edited owned files and unowned destination collisions prevent replacement: preserve changes and agree on migration, or inspect candidates in a fresh output root.

`.openapi-kt.lock` is persistent; existence alone does not mean busy. For `E_OUTPUT_BUSY`, let the actual writer finish and retry, retaining the lock file. For `E_RECOVERY_REQUIRED`, stop writes and preserve `.openapi-kt-stage-*`, its `transaction.json` and backups for an explicitly approved manual recovery. There is no automatic recovery command or `--force` bypass.

Ordinary recoverable commit failures roll back; power-loss or multi-filesystem crash atomicity is not promised. If rollback fails, retain backups. Removing manifests, locks, backups or output roots is not a remedy for a safety error. Obtain confirmation before moving/restoring user files or attempting recovery.

## Build from source

Prefer the selected executable; inspect its version/help when compatibility is uncertain. If a source build is needed, locate the checkout containing `go.mod` and `cmd/openapi-kt`; ask for its location only if it is unavailable. Read `go.mod` to select an installed toolchain meeting that checkout's minimum Go version; a preview tag may require an older version than the development branch:

The configuration and document adapters use dependencies pinned by `go.mod`/`go.sum` and included in `vendor/`. Ordinary offline source builds require no module-cache preparation.

```bash
# Installed Go matching this checkout's go.mod and GOROOT, with prepared modules; no downloads.
mkdir -p build
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -trimpath -o build/openapi-kt ./cmd/openapi-kt
build/openapi-kt --version
```

Use the explicit binary path in subsequent commands. Consumers need no Go runtime. Correct a mismatched `GOROOT` by selecting an existing matching toolchain for the current process rather than changing global configuration.

### Selected-owner root move

For existing v2/v4 named output, add `--spec NAME` to the explicit root-move command. Supply the complete original single-root config and unchanged inputs/options: this migration reads every original input to bind exact Kotlin and lock semantics. The selected owner moves to an empty or [compatible owned target](#owned-target-root-merge); peers and their shared payloads stay at source. Review `moved_owners`, `retained_owners`, selected file/lock hashes and the original full-source manifest hash. The split rewrites manifest membership at the same version, preserving Kotlin/raw-lock bytes, modes and mtimes. A source or retained-peer change rejects under the same whole-source CAS/locks/rollback contract.

After success, change only that spec's output_dir, retain peers at source, drop the action and run ordinary full config `--check`. Include shared Kotlin declarations once in a compilation. Explicit runtime_package owners must remain in one root; use a complete-root move for that inventory. Arbitrary owner subsets, incomplete source inventories, conflicting target payloads and adoption retain their separate restrictions.

### Configured warning policy

Defaults and specs accept `deny_warnings` plus exact-code `allow_warnings`; false/empty replace defaults. A nonempty allowlist requires effective deny mode, and malformed codes fail before input/state reads. `--deny-warnings` enables every generated spec; explicit CLI allow codes replace configured lists, while an omitted CLI list preserves them. CLI allow syntax still requires CLI deny. Review all original warnings and rejected original Pointers; denial occurs after full in-memory generation and before any output change. The same effective policy covers rename/root move/adoption/lock actions, using each action's original source-control inventory.

### Disjoint configuration root moves

One explicit source/target pair may be selected within a config containing other disjoint roots. Supply the complete original source-owner group; optional `--spec` chooses one source owner. The action generates that source group only. Other roots' inputs and owned state are untouched, but declared canonical ancestor bindings validate before source input reads and again in preflight/apply. Unknown extra owners at moved roots, missing source peers and overlapping roots refuse. [Same-version v2/v4 owned targets](#owned-target-root-merge) accept disjoint moved owner names and byte-identical shared Kotlin paths: original target owners may remain declared at TARGET, their inputs are not read, and complete target state participates in pre-input capture and CAS. Review target_manifest_sha256 and target_owners along with the original source report. Update moved specs afterward, retain unrelated roots, and use ordinary config `--check` with its normal inputs.

Use `bash scripts/test.sh feature composed-member-defaults` in the generator repository for fresh Cartesian enum-member defaults, numeric selection migration, four Json configurations and HTTP/SSE/Patch consumers.

Optional raw text/octet request parameters default to null for absence, including SSE; non-null empty text/bytes retain raw Content-Type. Schema nullable remains unsupported. Use `feature raw-request-media` for the combined required/optional wire contract.

`bash scripts/test.sh feature form-request-media` includes optional body omission, present/empty scalar fields, HTTP/SSE and combined optimized/locked output. Source field grammar and constructor limits retain their existing bounds.

With validation enabled, existing ordinary allOf object default roles admit complete repeated plain String constraint groups. Earlier own annotations stay local; winning and supplied outer values satisfy the retained conjunction. Missing/required/Patch/source/budget/codec contracts remain. Use `feature allof-string-defaults` for Data/Plain/fourJson/optimized/width240/HTTP/SSE checks; numeric/ref/enum/unchecked duplicate roles remain separate.

Eligible finite inline List/typed Map component roots with ordinary DTO, direct named plain DTO or bare ordinary allOf leaves can own opt-in top-level `<Alias> default` factories. Use the actual emitted alias/serializer; referencing fields do not inherit root defaults. Existing nullable, required/Patch, own-property source and budget rules remain. `feature object-collection-factories` verifies four fresh profiles, optimized/locked output, wide/deep and HTTP/SSE consumers; element alias/internal/union/hierarchy and child-default roles stay separate.

Numeric enum component-own factories also accept no format, retaining Int (`integer`) or Double (`number`). Defaults strictly match original vocabulary/kind before rounding; nullable native roots/null members retain validation-on admission, and root factories do not initialize referencing fields. Use `feature unformatted-native-factories` for four profiles, optimized/locked output, IEEE/negative-zero/extrema/240-members+slots and HTTP/Patch/SSE consumers.

With validation, ordinary allOf outer defaults can consume complete repeated plain Int/Long/Float/Double groups already admitted by primitive aggregation. Integer checks retain exact original bounds/factors; Float/Double keep mapped comparisons and floating multipleOf-default exclusions. Original own annotations validate before override, the winning annotation satisfies the conjunction, and omitted members keep actual constructor defaults. Required/Patch/no-inheritance and resource caps remain; `feature allof-numeric-defaults` verifies Data/Plain, optimized/locked output, four Json settings and HTTP/Patch/SSE.

OpenAPI 3.1 direct scalar prefixItems component roots may own independent List default factories, including pure component aliases. Uniform prefixes or admitted scalar items tails retain their existing List type/codec and approximation; original prefix sources validate before empty/null defaults. Root factories are not inherited by fields; nominal/raw/member-own tuple factory roles remain separate. Use `feature tuple-root-defaults` for Data/Plain×validation, optimized/locked output, four Json settings, temporal/IEEE/Long/240field and HTTP/Patch/SSE consumers.

Selected flat form DTOs also accept optional non-null builtin fields: Kotlin nullable defaults mean omitted form keys, retaining empty Strings/false/zero values and original property order/wire names. A supplied all-null DTO sends an empty form with media, distinct from an absent optional body. Explicit schema-null/default/reference/Patch states stay excluded; use `feature optional-form-fields` for four profiles, optimized output, 240 fields and HTTP/live SSE consumers.

Flat form fields also accept direct Float/Double, including unformatted number→Double, using mapped runtime scalar toString text and Ktor UTF-8 encoding. Negative zero/exponent/rounding and unconstrained NaN/Infinity text remain; admitted numeric constraints retain opt-in finite/bounds checks. Optional null omits a key; schema-null/default/caller-native/Patch roles remain excluded. `feature floating-form-fields` checks four profiles, optimized output, Double JVM240slots and HTTP/live SSE.

Direct plain String/native scalar enum form fields retain actual local String declarations and mapped builtin types. Encode String serialName (including Unrecognized), native scalar text with opt-in membership/checks, optional null omission and independent body absence. Selected raw vocabularies cap240 under both validation modes; formatted String/ref/alias/default/schema-null/Patch roles remain separate. Optimized DTO aliases use the actual retained owner for nested enum access. `feature enum-form-fields` verifies four profiles, actualowner/240vocab, Long/IEEE and HTTP/live SSE.

Large repeated collection traversal checks may share private typed methods after a 12KiB inline-source planning threshold. Public types/constructor/copy/serde/checks/wire and persisted formats remain; no state/casts or schema cap changes. `feature wide-deep-validation` checks List<Int>200×16 and240×12 in Data/Plain×8optimization combinations, factory/copy/Json/HTTP/SSE, with complete 128KiB compilation partitions and existing heap. JVM UTF8 generic-signature limits remain separate: do not claim full width×depth Cartesian or cross-platform acceptance from those profiles.

Retained DTO constructors whose guaranteed minimum generic-signature bytes exceed JVM65535 UTF8 capacity fail with E_MODEL_LIMIT and original component/nested/HTTP schema Pointer before output. Known inline containers/builtin boxing only are counted; unknown aliases/names/wildcards are omitted, so passing is not a complete backend guarantee. Feasible wide-deep profiles/public types/wire/caps remain. Native generate/check/validate preserve old output. See `docs/jvm-signature-limits.md`; broad exact-signature and cross-platform work remains separate.

Selected flat form DTOs also accept direct UUID/date/date-time fields using existing Uuid/LocalDate/Instant types and canonical toString text through Ktor UTF-8 encoding. Optional field null omits its key; optional body null omits content. UUID checks remain validation opt-in, temporal bounds/ref/default/schema-null/native/formatted-enum roles stay excluded, and existing budgets hold. See [standard form fields](../../../docs/standard-form-fields.md); run `bash scripts/test.sh feature standard-form-fields` for fresh JVM profiles.

Validation-enabled ordinary allOf outer defaults also support complete repeated direct UUID fields already eligible for retained primitive checks. Length/pattern checks use canonical Uuid text, original own/winning/outer annotations validate with their source Pointers, and required/Patch/no-inheritance/shared budgets remain. See [allOf UUID defaults](../../../docs/allof-uuid-defaults.md); `feature allof-uuid-defaults` is the fresh JVM route. Other source/format/enum/reference roles remain separate.

The JVM constructor lower-bound guard also counts demonstrated variance bytes for entire direct builtin List chains. It rejects204×List16 while retaining compiled203×16 Data/Plain; uncertain alias/Map/mixed/nullable-element/copy/name metadata remains outside this narrow proof. Run `feature jvm-signature-limits`; see [known List variance limits](../../../docs/jvm-list-variance-limits.md). Public types, source Pointers, existing caps and old-output preservation remain.

A post-transform signature guard counts the known parameter lower bound plus the ASCII package prefix of retained Data copy return types. Final Plain models are exempt from the Data copy proof. Failures retain original Pointers and old output; diagnostic Plan metadata grants no source permission or shape/lock identity. See [Data copy signature limits](../../../docs/jvm-copy-signature-limits.md), and run `feature jvm-copy-signatures`. Exact aliases/names/other metadata and wider platform feasibility remain separate.

Already-bound root bridges through internal chains ending at original direct named non-null ordinary DTOs can contribute an allOf shape. Fields keep original nodes/Pointers, real catalog/owners, defaults/checks, required/Patch and full cached source height. Nullable/composed/internal lexical/union/hierarchy/count/raw targets remain excluded from this role; existing caps and failure preservation hold. See [bridge intersections](../../../docs/bridge-intersections.md), and run `feature bridge-intersections`. Other nominal and typed bridge contexts retain separate contracts.

The named ordinary DTO root-bridge allOf role also accepts intrinsic target nullable intent and nearest alias/member true/false overrides. Discovery traverses original ref nodes without granting ownership/annotation authority; existing source/work/depth/slot caps and final catalog validation hold. Use canonical non-null companion codecs or nullable adapters according to the typed view. See [nullable bridge intersections](../../../docs/nullable-bridge-intersections.md), and run `feature nullable-bridge-intersections`; other endpoint and shape roles remain separate.

AllOf can borrow fields from actual ordinary internal DTOs under direct ordinary owners through direct/finite-chain/root-bridge refs. Original nodes/Pointers/canonical owners and final unique-constructor proof remain; no internal component identity is synthesized. A shared16384work/8MiB capture budget charges distinct overlapping targets; existing source/checked/slot/default caps, nullable/Patch and failure preservation hold. Composed/hierarchy/count/root-default/scratch roles remain excluded. See [internal DTO intersections](../../../docs/internal-dto-intersections.md), and run `feature internal-dto-intersections` for fresh nullable/non-null Data/Plain profiles.

AllOf may also borrow a unique actual ordinary nested property/collection DTO under an allOf/union lexical owner. Original source/canonical owner and deferred unique-occurrence proof remain; variant roots/common/hierarchy/count/root-default/ambiguous/scratch roles stay excluded. Existing shared capture and source/checked/slot caps, defaults/Patch/wire and failure preservation hold. See [composed-owned DTO intersections](../../../docs/composed-dto-intersections.md), and run `feature composed-dto-intersections`; construct canonical nested types according to actual generated aliases.

AllOf may also borrow an actual ordinary nested DTO directly owned by a named plain-source hierarchy leaf, via internal chains/root bridges. Original unique-owner/capture proof, nullable/Patch/default/recursion and all prior caps remain; parent kind/common fields are not part of the nested schema or Consumer slots. Root/common/inline/composed-leaf/count/root-default/ambiguous roles remain separate. See [hierarchy-leaf nested DTO intersections](../../../docs/hierarchy-dto-intersections.md), and run `feature hierarchy-dto-intersections`. Existing dispatch and persisted formats are unchanged.

`bash scripts/test.sh feature composed-leaf-intersections` verifies unique original nested ordinary DTO operands under named composed hierarchy leaves with fresh Kotlin API/JSON/Patch/HTTP/SSE profiles.

`bash scripts/test.sh feature allof-target-intersections` verifies nested ordinary allOf operands under direct ordinary object owners with fresh Kotlin API/JSON/Patch/HTTP/SSE profiles.

`bash scripts/test.sh feature composed-target-owners` verifies four original lexical owner roles with sixteen fresh Kotlin API/JSON/Patch/HTTP/SSE profiles and combined optimized/locked outputs.

`bash scripts/test.sh feature allof-directional-models` verifies ordinary allOf request/response projection with fresh Kotlin call sites, four Json settings, Patch/recursion/optimized HTTP/SSE and a240-field request constructor.

`bash scripts/test.sh feature dto-unique` verifies mapped scalar-field DTO uniqueness, constructor-aware defaults, named/inline/nullable keys, baseline/optimized locks, HTTP/SSE/Patch and independent240×240 Kotlin/resource profiles.

## Existing-v4 donor adoption

For an existing locked v4 inventory, select the new spec and add `--adopt-locked-owner-from DONOR` to the adoption commands. The donor must exist in the same effective root; the input must reproduce an exact donor Kotlin/allocation-occurrence subset with its lock profile. Old peers need not be configured and retain memberships/raw locks. Check3 proposes metadata addition; safe validate0 is write-free; exact replay returns check0. Review `from_owner`, donor `source_symbols`, selected files/symbols and manifest/raw-lock hashes. Ordinary named generation manages later changes. New allocations/occurrences, changed bytes, peer-only files and ownership redistribution/removal remain separate.

### Owned-target root merge

The existing explicit `--move-output-from SOURCE --output TARGET` accepts same-version v2/v4 named targets for full or single-source-owner selection. Moved owner names must be disjoint from target. An exact existing target-owned Kotlin path may be shared only when original bytes match; keep target mode/mtime and list it in shared_target_files. Different bytes, case-only paths and unowned destinations still refuse. Complete original source generation remains required; all existing target Kotlin/raw locks/attributes are captured before source inputs and rechecked in both readonly and locked planning. Original target owners may remain configured at exactly TARGET without their inputs being read; other declared peers retain disjoint/stable binding requirements.

Review original source manifest and moved payload hashes plus `target_manifest_sha256` and sorted `target_owners`. The same-version union rewrites membership manifests only, preserving new payload source bytes/modes/mtimes and every existing target peer/raw lock, while shared existing target Kotlin retains target attributes. Combined target metadata must still fit original 256-lock/16MiB aggregate, per-lock and manifest limits. Both roots stage under sorted locks before apply, each manifest is last, and ordinary failures roll both back; recovery evidence and crash limits remain. Update moved output_dir values, drop the one-shot flag and run ordinary full config --check with normal inputs. Conflicting target payloads still refuse; owner redistribution/removal, version conversion and adoption remain separate.

### Reassign locked owner memberships

Use `generate|validate --config FILE --reassign-locked-owners` for an existing nonempty fully locked v4 root with one original package/suffix/qualified profile. Supply the complete desired single-root inventory with effective hoisting/locking and no --spec. Owner IDs may be added/removed, but every original Kotlin path/byte and original role/shape/public name/occurrence must remain covered. Existing projection preserves historical names and retained singletons; reset is unnecessary. Existing ordinary --reconcile already handles omitted-owner cleanup, and rename/donor actions keep their contracts.

The command captures all original owned bytes/modes/mtimes, canonical root binding and physical identity before desired inputs; every old peer, including omitted ones, participates in locked CAS. Union limits stay1MiB/4096entries/16384occurrences,256owners,64MiB original and aggregate candidate Kotlin,16MiB actual raw locks and existing metadata limits. Unchanged semantic raw locks prefer same owner, then sorted original donors; exact donor copies retain donor attrs. Only owned lock/manifest metadata is staged, manifest last; Kotlin/unowned files stay in place. Normal rollback/recovery-evidence limits remain. New allocations, incomplete coverage, profile/name/origin conflicts and intervening state refuse.

Review locked_owner_reassignment with original/result manifest hashes, full old/new owners, complete unchanged file hashes, per-owner memberships/lock hashes and original/result public symbols. Check returns3 for safe change; validate/apply and exact raw replay return0. Drop the action and use desired ordinary config --check afterward. Value/duplicate syntax, partial selection, missing config and combinations with adoption/donor/reconcile/rename/root move/reset/disable refuse. No TOML key, root discovery, version conversion, Kotlin emitter or wire change is introduced.

### Retain original ownership during partial adoption

Use `generate|validate --config candidates.toml --adopt-locked-owner --retain-locked-owner NAME` with no --spec. On locked v3, actual direct code/raw lock becomes complete named NAME ownership; on v4, NAME must already own all root Kotlin and have its owned lock. Candidate names differ from NAME, share its package/suffix/qualified profile and use hoisting/locking. Every candidate Kotlin byte/path and allocation/name/occurrence is an exact original subset. NAME retains everything not assigned to a candidate, plus complete original ownership; existing peers are preserved and can only exact-replay under this action. This performs existing full adoption then donor addition as one metadata transaction.

Complete raw state/modes/mtimes/canonical binding/physical identity is captured before candidate inputs and rechecked under the root lock, including unselected peers. Actual original reads64MiB, retained original plus generated candidates64MiB, result owners/locks256, raw locks16MiB and old single-lock/metadata caps remain. Copied semantic raw payloads preserve donor attrs. Only owned metadata stages, manifest last; Kotlin/unowned files are untouched, normal rollback/recovery-evidence limits remain. Review retained_locked_adoption with actual versions/manifest hashes, complete retained files/raw lock/public symbols and candidate subsets. Changed check3, validate/apply0, exact v4 replay0; flag/identity/profile/coverage/source conflicts refuse.

Afterward, use ordinary --spec NAME for a candidate-only config. For ordinary full generation, list retained NAME with its original full input/options and all desired peers: omitted owners are reconciled. Direct v3 commands refuse v4 after adoption. Retain requires adopt and a concrete name, and rejects partial selection, donor-from, reassign, rename/move/reconcile/reset/disable combinations. No TOML key, source download, implicit ownership placeholder, Kotlin/wire or persisted-version extension is introduced.

A retained source-backed String enum member whose generated name alone exceeds 65,535 UTF-8 bytes now fails `E_MODEL_LIMIT` at its original enum member Pointer after transforms. Generation and `--check` keep old files/locks. Shorten the source token when this happens; names are preserved, and acceptance does not guarantee general JVM/path feasibility. See [enum JVM name limits](../../../docs/enum-jvm-name-limits.md). The existing `feature narrowed-enum-wrapper-defaults` route compiles same-byte `VALUE` controls and checks the original oversized slash-token refusal.
