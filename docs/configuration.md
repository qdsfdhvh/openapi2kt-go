# Local configuration

`generate` and `validate` accept `--config FILE` instead of `--input`. Generation remains offline. Configured inputs are local files only (`input = '-'` is rejected); direct single-document commands separately accept [stdin](local-input.md). Specification acquisition is external, not a built-in download command. This configuration envelope supports local JSON or [bounded YAML](yaml-input.md) specs in one or several **disjoint output roots**, with version 2 named-owner output and [normal-failure batch rollback](output-batches.md). Unlocked direct `--input` commands retain version 1 behavior. Opt-in [hoisted-name locking](name-lock.md#cli-and-configuration) uses version3 direct/version4 named-owner metadata with the same normal-failure transactions. It does not implement the complete roadmap configuration surface.

```toml
config_version = 1
[defaults]
output_dir = "generated"
package_name = "org.example.api"
model_name_suffix = "Dto"
validation = true
[[specs]]
name = "catalog"
input = "catalog.json"
[[specs]]
name = "events"
input = "events.json"
validation = false
```

```sh
openapi-kt generate --config openapi-kt.toml
openapi-kt generate --config openapi-kt.toml --spec catalog
openapi-kt generate --config openapi-kt.toml --check
openapi-kt validate --config openapi-kt.toml --spec catalog
```

## Schema and precedence

The root requires integer `config_version = 1`. `defaults` is an optional table and `specs` is an optional array of tables; equivalent `[defaults]` / repeated `[[specs]]`, dotted keys, and inline table/array forms are supported. Removed `specs.download` tables are rejected with `E_CONFIG_UNKNOWN_FIELD`. Each spec requires a unique, case-sensitive ASCII `name` beginning with a letter and continuing with letters, digits, `_` or `-`. Effective `input`, `package_name` and `output_dir` are required. `validate` supplies the same builtin package default as its direct command, but still requires configured output layout. At most 256 specs and 1 MiB configuration input are accepted; existing generator schema/collection/constructor budgets are unchanged. Additional configuration syntax budgets are defined below.

Defaults and specs accept: `input`, `input_format` (`auto`, `json`, `yaml`), `output_dir`, `package_name`, `model_name_suffix`, `api_client_name`, `runtime_package`, `model_style` (empty/`data_class`/`plain_class`), `exclude_apis`, `exclude_merge_models`, `exclude_merge_model_patterns` and `exclude_merge_model_suffixes` (string arrays), `merge_identical_models`, `deduplicate_models`, `hoist_shared_shapes`, `qualify_hoisted_names`, `lock_hoisted_names` (booleans, default false), `deny_warnings` (boolean, default false), `allow_warnings` (exact warning-code string array, default empty), and `validation` (boolean). YAML uses the same analyzer/output pipeline under the [YAML input profile](yaml-input.md); `auto` recognizes `.yaml`/`.yml` and explicit `yaml` works for configured local files.

Precedence is builtins < defaults < spec < explicitly provided CLI options. Empty arrays/strings and `false` replace defaults, rather than appending or inheriting. [Initial shared-root hoisting](hoist-models.md) relocates repeated ordinary DTO/enum roots with role/owner-qualified names and exported API-root aliases; qualification and name locking require effective hoisting. The dependency is checked across the entire effective inventory before selected input/state reads. Stored locks cannot implicitly be disabled by setting false; use the invocation-only [disable/reset lifecycle](name-lock.md#explicit-disablereset-lifecycle) action with the intended final locking mode and review its migration report. Unselected locked owners are retained. [Initial operation DTO/enum deduplication](deduplicate-models.md) preserves exported root aliases but intentionally changes canonical class/descriptor identity and nested-classifier access. [Plain model style](model-style.md) retains wire/constructors/getters but intentionally removes copy/componentN. Generation option CLI overrides require one selected spec (or a one-spec inventory); multiple-spec overrides fail with `E_CONFIG_OVERRIDE`. `--exclude-api` replaces the configured list. [Initial resolved model merging](merge-models.md) is default off; exact fully qualified `exclude_merge_models` roots neither absorb nor are absorbed and require effective merging. `--exclude-merge-model` replaces that list; false/empty overrides remain explicit. [Pattern and suffix exclusions](merge-exclusions.md) use independent lists and corresponding repeatable `--exclude-merge-model-pattern`/`--exclude-merge-model-suffix` flags, replacing only their own effective list. Dependency/name validation covers the entire inventory before selected input/state IO. Configured paths are relative to the configuration file's directory, independently of the current working directory; absolute paths remain absolute. Explicit `--output` paths retain the ordinary CLI current-directory interpretation. Participating roots must be disjoint and non-aliasing. An explicit output override selects the destination only; relocating an output root does not automatically clean a former root or migrate its ownership. The separate invocation-only `--move-output-from SOURCE --output TARGET` action requires complete existing ownership: direct v1/v3 or full same-root config v2/v4, every original owner/input/options, exact Kotlin/lock semantics and no reconciliation. For that action only, full inventory may explicitly override output; declared config root must canonically match source or target. See [root move](output-root-move.md) for read-only check/validate, raw bytes/attributes and sorted two-root CAS/rollback. Selected-owner and disjoint-source-group profiles are described in that contract; missing source peers, adoption, owner rename and lock actions still refuse.

Unknown fields/sections, duplicate declarations/names and invalid types fail. Configuration diagnostics use synthetic JSON Pointers identifying a config location; they do not quote arbitrary input values. Schema errors retain their original schema Pointer, with an additive report `spec` identifier. Successful config reports have additive per-spec `specs` summaries and sorted selected generated paths (one entry per distinct root/path; relative paths may repeat across roots). Warning policy accepts the existing CLI `--deny-warnings`/`--allow-warning` flags; warnings retain their per-spec provenance. Default direct-command reports are unchanged.

## Supported TOML language

The configuration adapter uses the pinned, MIT-licensed [`go-toml/v2`](https://github.com/pelletier/go-toml) dependency for **TOML 1.1** syntax. Bare, basic-quoted and literal-quoted keys refer to the same decoded spelling; dots inside quotes are part of one key, while dots outside quotes create a table path. Basic/literal and multiline strings, escapes (including TOML 1.1 `\xHH` and `\e`), line continuation, inline tables, multiline arrays with comments/trailing commas, and arrays of tables are parsed by the library. Inline-table newlines/comments/trailing commas follow TOML 1.1. LF/CRLF are supported. Invalid UTF-8, BOM, raw forbidden controls, invalid escapes/Unicode scalars, malformed number/date tokens and duplicate or conflicting key/table declarations fail with safe diagnostics.

Syntax does not expand the schema. Only the fields and value types listed above are accepted. Strings remain strings without interpolation, environment expansion or coercion; an escaped/multiline path uses its decoded bytes and the same config-relative path rules. `config_version` must be integer value 1: `+1`, `0x0_1`, `0o1` and `0b1` are equivalent spellings; other integers fail with `E_CONFIG_VERSION`. TOML floats (including `inf`/`nan`), local dates/times and offset datetimes are parsed, then rejected with `E_CONFIG_INVALID` and the required field type where an integer/string/boolean/string array is expected. Heterogeneous/nested arrays and tables in scalar fields also fail. Unknown keys/tables, download metadata and plugin DSL remain rejected. Configured `input = '-'` is still invalid, including in defaults and unselected specs. An empty inventory may omit `specs` or use `specs = []`.

This is an equivalent spelling of the first example:

```toml
"config_version" = 0x0_1
'defaults'.output_dir = """
gen\
  erated"""
defaults.package_name = 'org.example.api'
defaults.model_name_suffix = 'Dto'
defaults.validation = true
specs = [
  { name = 'catalog', input = 'catalog.json' },
  { name = 'events', input = 'events.json', validation = false },
]
```

Configuration parsing has fixed limits in addition to 1 MiB and 256 specs: **64 structural path/container levels**, **65,536 AST nodes** (including key parts) and **33,554,432 work units**. Work counts node visits plus conservative duplicate-tracker costs: cumulative key comparisons, comparisons against anonymous array-container entries, and array-table inventory scans/remaps. An iterative AST preflight enforces these limits before map decoding; the library also caps initial value parsing at 10,000 nested containers. Exceeding a budget returns `E_CONFIG_LIMIT`; oversized/deep/wide syntax cannot invoke unbounded adapter recursion. These limits also apply to rejected/unknown values. Budget failures take precedence over schema checks.

The closed schema visits keys deterministically in source order within each table and preserves explicit empty/false values. Duplicate spellings fail with `E_CONFIG_DUPLICATE`. Diagnostics identify known config fields or the closest known container, with a spec index when available; duplicates inside an inline table point to the enclosing assignment. Arbitrary unknown key names, values, parser messages and source excerpts are never included in reports. Newly admitted syntax needs no config/manifest/name-lock version migration; existing accepted configurations produce the same effective options, output bytes and ownership behavior. This completes the TOML syntax increment of D10, not the whole F02 configuration/lifecycle roadmap.

For source builds, the pinned parser source is included in `vendor/`; ordinary offline checks require no module-cache preparation. See [installation](installation.md) and [dependency notices](../THIRD-PARTY-NOTICES). The dependency is confined to the configuration adapter; the generation core remains free of configuration IO.

## Ownership and failure safety

All selected specs generate successfully in memory before any output write; all participating roots then preflight and stage before application, with all cooperative locks held until success or rollback. A later selected spec's input/schema failure or warning-policy rejection preserves previous owned/handwritten output. Partial `--spec` generation does not read unselected inputs and preserves their owner entries, even when an owner was removed from the current inventory. Full ordinary generation reconciles owners absent from the complete inventory; explicit [owner rename](output-owner-rename.md) retains them without reconciliation. Shared paths require identical bytes; the final owner reference controls deletion. See [output ownership](output-ownership.md) for locking, rollback, missing retained files, no-op/check and version protections.

An empty inventory requires explicit `generate --config FILE --reconcile` and an effective default `output_dir`. `--check` reports differences without creating output directories/locks or modifying files. Empty reconciliation cleans manifest-owned files only. A selected empty/missing spec name is an error; `--reconcile` cannot be combined with `--spec` or `--rename-owner`.

No automatic version 1 adoption/migration is implemented. Named-owner writes refuse legacy trees with `E_MANIFEST_OWNER_MIGRATION` by default. [Explicit exact-match metadata-only migration](ownership-migration.md) is available through `generate --config FILE --spec NAME --adopt-legacy`; preserve the original document/options and authorize the selected owner before applying it. Moving/renaming output roots does not clean old roots. [Shared runtime_package](runtime-package.md) relocates existing Patch/bigint helpers with same-package/same-root ownership; it does not relocate all private helpers or create a standalone runtime library. Cross-root crash atomicity/automatic recovery and general hoisting/merging remain unimplemented. [Explicit complete output-root move](output-root-move.md) supports one concrete source/target pair with unchanged v1–v4 bytes and ownership; selected-owner moves use the complete original source inventory, including within a disjoint configuration; [same-version named owned-target merging](output-root-merge.md) accepts disjoint moved owner identities and exact bytes for shared Kotlin paths with complete both-side capture/CAS; same-root v3-to-v4 adoption uses the separate locked-owner action below.

An explicit [locked-owner adoption](ownership-migration.md#locked-owner-adoption) workflow on `generate|validate --config FILE --spec NAME --adopt-locked-owner` assigns an existing complete direct v3 root to exactly one configured locked spec in the same root. It preserves the entire Kotlin inventory and raw owned lock, uses pre-input canonical root/manifest/Kotlin/lock binding and repeats exact CAS in read-only preflight and under sorted locks. Apply journals only the raw lock filename transfer and v4 manifest, with manifest last. Configuration must continue using that named owner afterward; ordinary direct v3 commands refuse v4. Exact same-owner complete v4 replay is a validated no-op, not ordinary updates. Fresh/unlocked roots, multi-spec/partial adoption and combinations with legacy adoption, root move, rename, reset/disable or reconcile refuse. `--adopt-legacy` retains its existing v1 behavior.

[Explicit named-owner rename](output-owner-rename.md) uses repeatable, invocation-only `--rename-owner OLD:NEW` on `generate|validate --config`. `NEW` is a selected current spec; `OLD` is its existing version2/4 owner in the same effective root. Full selection maps every current spec, with identity mappings for unchanged names; partial `--spec NEW` requires exactly one matching mapping. There is no TOML rename key or root discovery/move; v3→v4 uses separate locked-owner adoption. Rename preserves exact Kotlin membership/bytes and lock mode/profile/allocations/occurrences, retains all peers and shared memberships even when absent from config, and never reconciles. Check/rename validation are read-only; apply uses exact manifest plus all owned lock-byte CAS under the complete sorted lock set and a metadata-only journal with each manifest last. Review additive `owner_renames` data and point-in-time impact guidance; identity is a validated no-op. Rename cannot be combined with `--adopt-legacy`, `--adopt-locked-owner`, `--reconcile` or `--name-lock-action`.

Explicit existing-state `--name-lock-action reset|disable` is available on generate/validate, including partial selection and multi-root transactions; it requires matching stored hoist/package/suffix/qualification profiles, existing valid state for every selected spec, and effective locking true for reset/false for disable. Full selection applies the action to all selected specs; action mode mismatches fail before input/state IO. There is no TOML action key; keep only the final `lock_hoisted_names` setting in configuration. Review `--check --diagnostic-format json` for public classifier/descriptor and Kotlin file-hash differences. Check and action validation are read-only; selected state uses exact-byte CAS under the sorted lock set. Disable retains version4, unselected owners/locks and shared runtime references; full reconciliation still removes only owned absent inventory entries. Actions cannot be combined with `--adopt-legacy`, `--adopt-locked-owner`, `--rename-owner` or empty `--reconcile`; version3-to-version4 uses the separate exact locked-owner adoption contract. Exact, portable pattern and literal suffix generated-root merge exclusions belong to the merging profile above. Remove old download tables and fetch inputs externally before local configuration generation. `input = './-'` refers to a literal local dash file; no configured stdin marker/replay exists. See [breaking CLI/config migration](local-input.md#configuration-and-migration); config/output/name-lock versions and generated APIs are unchanged. [Configuration `--metrics` aggregation](cli-metrics.md#configuration-aggregation) is available: config bytes are separate, selected source reads/phase durations accumulate, and complete unambiguous planned candidates deduplicate shared files per root/path. It includes nested core boundary durations and available process-lifetime peak RSS; it does not attribute resolve/name costs or memory to phases and is not a benchmark.

Verification includes [grammar/schema diagnostics](../internal/config/full_toml_test.go), [independent resource budgets](../internal/config/toml_budget_test.go), [native TOML parity/failure preservation/reconciliation](../internal/cli/full_toml_config_process_test.go), [decoder tests/fuzz](../internal/config/config_test.go), [precedence/parser tests](../internal/cli/config_test.go), actual [native selection](../internal/cli/config_process_test.go), [failure/empty-reconcile guards](../internal/cli/config_guards_process_test.go), [whole-canonical source parity/reconciliation/migration/warning policy](../internal/cli/config_parity_process_test.go) and [Unix held-lock/check](../internal/cli/config_unix_process_test.go). The local config transport itself leaves emission decisions to core options, including opt-in runtime/package/model-style settings. The canonical parity test compares actual selected-union source bytes against direct generation of the combined schema.

For [selected-owner root moves](output-root-move.md#selected-owner-moves), `--spec` chooses ownership to move but the complete original same-root configuration is generated for byte/lock preservation control. Ordinary partial generation keeps its existing no-read behavior for unselected inputs. After moving, set the selected spec output_dir to the target and verify ordinary config `--check`; explicit shared runtime packages retain their same-root rule.

## Configured warning policy

`deny_warnings` and `allow_warnings` use defaults < spec precedence. Explicit false and empty arrays replace defaults. A nonempty effective allowlist requires effective denial; each entry must be an exact ASCII `W_` identifier under the [warning policy](warning-policy.md). Invalid effective policy refuses before input/state reads. Unknown well-formed codes are allowed for forward compatibility.

Invocation `--deny-warnings` enables denial for every generated spec; an explicit CLI allowlist replaces each configured list. Without a CLI allowlist, configured lists remain. These policy flags apply to the entire generated inventory and keep their existing CLI syntax (`--allow-warning` still requires `--deny-warnings`). Ordinary emission overrides retain their existing selected-spec restrictions.

Policy is evaluated per spec after complete successful in-memory generation, before comparison/staging/locks/writes. Warnings remain visible, and rejected warnings produce `E_WARNING_DENIED` in original spec/source order with original Pointers. A rejected spec preserves the entire output batch. Ordinary partial generation reads only the selected input; metadata migrations use their declared source-control inventory, including all inputs for root moves. Rename, root move, locked-owner adoption and explicit lock lifecycle actions use the same effective policy before applying their transactions. Read-only prior-state failures retain their original diagnostic precedence.

```toml
config_version = 1

[defaults]
deny_warnings = true
allow_warnings = ["W_ALL_OF_APPROXIMATION"]

[[specs]]
name = "catalog"
input = "catalog.json"
output_dir = "generated"
package_name = "org.example.catalog"
# A spec can replace the defaults:
allow_warnings = []
```

This additive configuration capability keeps config_version=1 and existing absent-field behavior. It changes acceptance policy only; artifacts, Kotlin/wire, output manifests and name-lock formats remain unchanged. The [schema checks](../internal/config/warning_policy_test.go) and [native ordinary/migration preservation tests](../internal/cli/config_warning_policy_process_test.go) define this concrete D10/F02 increment. Other configuration requirements remain independent.

[Disjoint configuration root moves](output-root-move.md#source-groups-in-disjoint-configurations) select one explicit source group by its complete original owner inventory. Other disjoint config groups remain untouched and their input/owned-state files are not read. Their canonical root bindings still validate before source input reads and again during preflight/apply. Update only moved specs after success.

## Generation-option parity

Every current public `core.GenerationOptions` field has a configuration setting in the schema above. The same effective local document, input format and generation options produce the same Kotlin artifacts and canonical name-lock payloads through core, direct CLI or named configuration. Ownership still differs deliberately: fresh direct output uses manifest1/3, while named output uses manifest2/4 and owner-specific lock filenames. These metadata differences do not change Kotlin or the lock profile/allocation semantics.

Defaults and spec settings share the same option vocabulary. Explicit false, empty strings where supported and empty arrays replace inherited values; they are not missing settings. This includes resetting Plain style/client/suffix/runtime choices and clearing API/merge exclusions and warning policy. A cleared validation flag rejects annotations requiring validation instead of silently inheriting true. `input_format` retains the local input contract: unknown extensions require an explicit supported format, while auto detection uses the documented file-name rules.

The [executable option parity contracts](../internal/cli/config_options_parity_process_test.go) compare all public option fields across core/direct/config, Data/Plain and defaults/spec placement. Separate controls exercise dedup, qualified hoist/locks, runtime/Patch relocation, each merge exclusion kind, explicit YAML, inherited false/empty resets, readonly/no-op behavior and failure preservation. Report formatting, metrics/check commands, caller-owned streams and explicit migration actions remain invocation concerns; previous lock bytes remain owned adapter state, and projection remains explicit migration planning. No additional TOML action or acquisition key is implied. New configuration capabilities need their own concrete schema and compatibility contract.

Existing version4 inventories can add one selected owner with invocation-only `--adopt-locked-owner --adopt-locked-owner-from DONOR --spec NEW`. The selected spec reproduces an unchanged donor Kotlin/allocation-occurrence subset in the same root; all old memberships/raw locks stay intact. Complete pre-input capture, donor profile validation, all-peer CAS and metadata-only rollback apply. See [existing locked inventory adoption](ownership-migration.md#add-an-owner-to-an-existing-locked-inventory); there is no new configuration key.

The invocation-only [locked owner reassignment](locked-owner-reassignment.md) action uses `--reassign-locked-owners` with a full desired single-root configuration and no `--spec`. Existing nonempty v4 must be fully locked with a compatible common profile; desired owners retain original names/occurrence union and complete Kotlin byte coverage. New/removed owner IDs change only metadata under complete original/root-identity/attribute CAS and manifest-last rollback. Warning policy applies before writes. It cannot combine with adoption/donor/reconcile/rename/root move/reset/disable and adds no TOML key.

[Partial adoption with retained source](retained-locked-adoption.md) adds invocation-only `--retain-locked-owner NAME` to `--adopt-locked-owner` with a nonempty same-root candidate configuration and no --spec. Retained NAME keeps complete original Kotlin and raw lock; candidates use exact projected subsets and existing peers remain. On v3 this creates named v4 ownership; on v4 retained NAME must already cover all root files. Afterward, a candidate-only ordinary update must select --spec; full ordinary generation includes retained NAME with its full original input/options because omitted owners are reconciled. No TOML key or generation-option override rule changes.
