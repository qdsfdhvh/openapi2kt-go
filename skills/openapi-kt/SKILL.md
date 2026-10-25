---
name: openapi-kt
description: Use when validating OpenAPI JSON or bounded YAML from local files or stdin with openapi-kt, generating Kotlin sources, adopting or splitting direct locked output among named owners, renaming configured output owners, explicitly moving owned output roots or selected configured owners, checking generated-source drift, or integrating the generated models and Ktor clients.
---

# openapi-kt

Generate kotlinx.serialization models and supported Ktor HTTP clients from a local OpenAPI JSON/YAML file or explicit stdin stream. Specification acquisition belongs to external tools; no built-in download or URL-fetch mode exists.

## Choose the task

Read only the reference section needed for the request:

| Request | Reference | Completion |
|---|---|---|
| Read a file or caller-supplied stream | [File or stdin](references/commands.md#file-or-stdin) | Use bounded JSON or the documented YAML profile; report upstream uncertainty and preserve output on input failure |
| Validate supported input | [Validate](references/commands.md#validate) | Report validation status and diagnostics; no writes |
| Generate or update sources | [Generate](references/commands.md#generate) | Review the emitted files and warnings; the same-options `--check` exits 0 |
| Reuse, disable or reset hoisted-name state | [Hoisted-name state](references/commands.md#hoisted-name-state) | Match the stored profile, review explicit lifecycle classifier/descriptor/file differences, and preserve owned state on failure |
| Adopt or split direct locked ownership | [Adopt a direct locked owner](references/commands.md#adopt-a-direct-locked-owner) | Verify complete same-root v3 coverage: one explicit spec or the full configuration with exact file/occurrence union; review projected/raw lock hashes and classifiers/descriptors, then confirm ordinary named config `--check` |
| Rename a configured output owner | [Rename a configured owner](references/commands.md#rename-a-configured-owner) | Verify same-root v2/v4 ownership and unchanged Kotlin/lock semantics, review `owner_renames`, then confirm ordinary generation under the new name with `--check` |
| Move an output root or selected configured owner | [Move an output root](references/commands.md#move-an-output-root) | Verify explicit source/target and complete original inventory, review preserved bytes/hashes/state, then confirm ordinary target generation with `--check` |
| Check local or CI drift | [Check drift](references/commands.md#check-drift) | Report consistency, differences or safety errors; no writes |
| Inspect nested String enum field defaults | [Nested String enum defaults](references/kotlin.md#nested-string-enum-collection-defaults) | Check the finite direct inline outer-field scope, preserve source diagnostics and presence, and report JVM evidence separately from PREPARE_ONLY |
| 检查嵌套 numeric/Boolean enum 字段默认值 | [Nested native enum defaults](references/kotlin.md#nested-native-enum-collection-defaults) | 核对 direct inline outer-own 角色、原 exact membership/presence/checks 和 native warning；分开报告 source 与实际 JVM 证据 |
| Inspect referenced String enum field defaults | [Referenced String enum defaults](references/kotlin.md#referenced-string-enum-collection-defaults) | Check the direct `$ref` field own-default role, strict original vocabulary, and report JVM evidence separately from PREPARE_ONLY |
| Inspect referenced String enum null-member defaults | [Referenced null-member defaults](references/kotlin.md#referenced-string-enum-null-member-defaults) | Check original explicit nullable intent, actual narrowed view and null versus String `"null"` |
| Inspect referenced String enum scalar defaults | [Referenced String scalar defaults](references/kotlin.md#referenced-string-enum-scalar-defaults) | Check field-own annotation, actual lexical classifier, strict vocabulary and open unknown Strings |
| Inspect referenced native enum scalar defaults | [Referenced native scalar defaults](references/kotlin.md#referenced-native-enum-scalar-defaults) | Check field-own annotation, existing reference admission and strict vocabulary before scalar mapping |
| 检查 referenced ordinary DTO object 字段默认值 | [Referenced object defaults](references/kotlin.md#referenced-ordinary-dto-object-field-defaults) | 核对 ordinary direct local `$ref` field own `default`、实际 nominal 类型/helper 绑定、nearest nullable 与 target-root 排除；JVM 证据与 PREPARE_ONLY 分开 |
| 检查 raw `JsonElement`/`JsonObject` 字段默认值 | [Raw JSON defaults](references/kotlin.md#raw-json-carrier-field-defaults) | 核对 open dictionary → `JsonObject`、unconstrained/empty/request-union → `JsonElement`、文件级 parse helper 与显式 null；typed dictionary/array/root default 仍排除 |
| 检查 inline primitive union 字段默认值 | [Union field defaults](references/kotlin.md#inline-primitive-union-field-defaults) | 核对 native primitive-union 声明、token 保留与 runtime selector wrapper、nullable/required/Patch、enum/const pin 生成期 membership 与 tag-dispatched object union variant 选择；非 tag/const-tag/presence/closed-subtype 与 request 上下文 union 仍排除 |
| 检查 referenced ordinary DTO 元素 collection 字段默认值 | [Referenced object collection defaults](references/kotlin.md#referenced-ordinary-dto-elements-in-collection-field-defaults) | 核对 inline `$ref` DTO 元素与 named collection `$ref` DTO 元素、helper 绑定与实际类型；element own/target-root default 仍排除 |
| 检查 inline List/Map of ordinary DTO object 字段默认值 | [Object collection defaults](references/kotlin.md#direct-inline-listtyped-map-of-ordinary-dto-object-field-defaults) | 核对 ordinary direct inline `array`/`Map` 字段 own `default`、元素对象递归、逐层 nullable 与 helper wrapper 绑定；element own default 与 referenced 元素仍排除 |
| 检查 inline ordinary DTO object 字段默认值 | [Nested object defaults](references/kotlin.md#direct-inline-ordinary-dto-object-field-defaults) | 核对 ordinary direct inline plain object own `default`、direct local `$ref`-valued properties、required/Patch、逐层 nullable 与 original Pointer；JVM 证据须与 PREPARE_ONLY 分开报告 |
| Inspect referenced native enum field defaults | [Referenced native enum defaults](references/kotlin.md#referenced-native-enum-collection-defaults) | Check scalar-wire vocabularies, exact default membership and validation bounds, and report JVM evidence separately from PREPARE_ONLY |
| Use positional arrays with enum references | [Enum tuple references](references/kotlin.md#positional-arrays-with-enum-references) | Check original vocabulary/nullability identity, approximation warning and physical enum APIs; compile fresh target sources |
| Merge primitive JSON-kind roots | [Primitive union merge](references/kotlin.md#merged-primitive-union-roots) | Review canonical classifier/descriptor/equality and overload migration; compile old wrapper/null alias calls against fresh sources |
| Use positional arrays with DTO references | [DTO tuple references](references/kotlin.md#positional-arrays-with-dto-references) | Verify original constructor/codec/nullability identity, approximation and source guards; compile fresh caller/serde behavior |
| Integrate models into Kotlin | [Sources and serialization](references/kotlin.md#gradle-sources-and-serialization) | Compile freshly generated sources in the consumer project |
| Use a generated HTTP client | [Ktor clients](references/kotlin.md#ktor-clients) | Compile and verify the relevant request/response behavior |
| Inspect errors or output conflicts | [Diagnostics and safety](references/commands.md#diagnostics-and-output-safety) | Explain the cause and safe next action; preserve existing files |
| Obtain an executable | [Source build](references/commands.md#build-from-source) | An existing or freshly built binary runs successfully |

Generation already validates input; a separate `validate` run is useful for a validation-only request, not a mandatory generation prerequisite. Supported operations automatically enable HTTP generation.

## Resolve inputs

Use the user's selected binary, supplied options and discoverable project configuration. Infer established input/output paths and naming conventions; ask only when unresolved choices affect the destination or public names. Keep checks consistent with generation options. Stdin is consumed once; a later validate/check needs a fresh stream or saved local file, not reuse of an exhausted pipe. Do not re-run an external producer without authorization.

## Essential boundaries

- Auto selects YAML only for `.yaml`/`.yml`; stdin/extensionless auto is JSON. For YAML streams use `--input-format yaml`. Follow the [input profile](references/commands.md#yaml-profile): preserve order, reject unsafe constructs and retain the original input when fixing them.
- Validation and drift-check requests remain read-only; report differences rather than automatically regenerating.
- Accept existing local files or explicitly supplied stdin; `--input -` is direct single-document only, not multi-spec config. Do not use removed `download` commands/tables, invent a curl wrapper, or acquire specs without explicit external-tool authorization. Keep credentials out of URLs/examples/results.
- Configured warning policy uses defaults/spec `deny_warnings` and exact `allow_warnings`; false/empty override defaults. Invocation policy flags apply to all generated sources. Review [configured policy](references/commands.md#configured-warning-policy) before CI checks or metadata migrations.
- `validate` checks this generator's supported subset, not complete OpenAPI or JSON Schema validity. Review approximation warnings even on success, and preserve unsupported semantics in the original spec.
- Local configuration uses bounded TOML 1.1 syntax for the existing closed `config_version`/`defaults`/`specs` schema. Equivalent quoted/dotted keys, multiline strings, inline tables and standard integer spellings preserve path/precedence/ownership behavior; types remain schema-defined. Follow [Inputs and options](references/commands.md#inputs-and-options) for config budgets and rejected metadata.
- [Retained-source partial locked adoption](references/commands.md#retain-original-ownership-during-partial-adoption) uses --adopt-locked-owner --retain-locked-owner NAME with candidate subsets and no --spec. NAME keeps all original Kotlin/raw lock; metadata-only projection/CAS preserves old peers. Later partial ordinary updates need --spec; full configuration must include retained ownership.
- [Locked owner reassignment](references/commands.md#reassign-locked-owner-memberships) explicitly projects complete original fully locked v4 ownership into a full desired common-profile inventory, including new/removed owner IDs. Preserve exact global Kotlin and historical allocation names/occurrence union, original caps and complete pre-input/root-identity/attribute CAS; only owned metadata changes.
- Direct output has one owner; configuration roots retain named-owner membership. Explicit owner rename preserves all peers and shared memberships, including peers absent from config, and uses a metadata-only transaction in the same root. Explicit complete root move preserves v1–v4 bytes/attributes and all original ownership; it requires a concrete source and target, complete original same-root source group and unchanged options. Configuration `--spec` moves one named owner while retaining peers, but still reads/generates the complete original inventory; explicit shared runtime packages retain their same-root rule. Other disjoint config groups remain untouched and their inputs/state are not read; their declared root bindings still validate. Same-version v2/v4 [owned-target merging](references/commands.md#owned-target-root-merge) accepts disjoint moved owner names and exact-byte shared Kotlin paths, retains target attributes for shared files, captures all target state before source inputs and preserves existing target peers through complete both-side CAS/rollback; explicit same-root locked-owner adoption supports complete direct v3→one named v4 owner or full-config split with exact original Kotlin and allocation-occurrence union. Changed lock subsets are projected with original names; unchanged payloads retain raw bytes. Keep the manifest and any owned name-lock metadata with generated files, and custom code outside generated files. Do not remove manifests, locks, recovery backups or the output root to bypass a safety error.

Report actual results and unverified items. Match verification to the task: CLI work needs no blanket Kotlin smoke run; integration uses the consumer project's relevant compilation/tests and dependency policy. Existing JVM smoke evidence does not verify a new Android/KMP target.

For direct enum fields in ordinary DTO `uniqueItems` arrays, use the [enum-field DTO contract](../../docs/dto-enum-unique.md). Validation is required; String enums compare serialName and native enums retain mapped membership/bounds.

For scalar const bodies in DTO `uniqueItems` arrays, see the [const-body contract](../../docs/dto-const-unique.md). Actual getter values, including decoded non-literal values, determine uniqueness; validation is required.

For direct JsonNull fields or null const bodies in DTO `uniqueItems` arrays, see the [JsonNull getter contract](../../docs/dto-null-unique.md). Validation is required; source/actual getter identity and existing null codec behavior remain authoritative.

For finite inline List/typed Map DTO getters with JsonNull leaves under validation, see the [JsonNull collection getter contract](../../docs/dto-null-collection-unique.md). Actual nullable getter states can collapse through the existing JSON codec; this grants no standalone JsonNull array uniqueness.

For finite inline List/typed Map DTO getters with existing numeric/Boolean enum leaves under validation, see the [native enum collection getter contract](../../docs/dto-native-collection-unique.md). Preserve original membership/bounds, mapped equality and nullable source rules; plain String enum collection/ref/const roles remain separate.

For finite inline List/typed Map DTO getters with plain String enum leaves under validation, see the [String enum collection getter contract](../../docs/dto-string-collection-unique.md). Keys use actual serialName; source defaults remain strict, while runtime unknown strings remain open. Ref/alias/format/bounded roles remain separate.

For ordinary DTO getters referencing already-supported builtin/temporal scalar paths under validation, see the [scalar reference getter contract](../../docs/dto-scalar-reference-unique.md). Keep original chain/declaration/type/initializer authority, nullable intent and field/target default roles; other reference getter families remain separate.

For ordinary DTO getters referencing already-supported numeric/Boolean enum scalar paths under validation, see the [native enum reference getter contract](../../docs/dto-native-reference-unique.md). Actual membership/check/export authority, nullable intent and field/target default roles remain; caller-native, const and other projections stay separate.

For ordinary DTO getters referencing already-supported plain String enum paths under validation, see the [String enum reference getter contract](../../docs/dto-string-reference-unique.md). Actual original enums/aliases and physical target types remain authoritative; annotations stay strict and runtime unknown serialName values stay open.

For inline builtin-scalar List/Map fields in DTO `uniqueItems` arrays, see the [collection getter contract](../../docs/dto-collection-unique.md). Validation is required; actual initializer/getter trees bind to original source and share existing work/depth budgets.

For null const properties inside admitted ordinary object defaults, see the [null const member contract](../../docs/null-const-object-defaults.md). Original null literals authorize this role; String "null" values and unadmitted default/reference roles remain rejected.

For ordinary JsonNull members and finite List/typed Map default leaves, including admitted component/wrapper collection factories, see the [JsonNull default contract](../../docs/null-value-defaults.md). Keep required/nullable/default roles separate; scalar component factories and new reference/const/uniqueness roles remain excluded.

For nonnullable JsonNull scalar/null-only enum component own default:null and independent component alias factories, see the [JsonNull component factory contract](../../docs/null-component-factories.md). Factories return the actual JsonNull singleton; consumers inherit no default, and nullable/const/format source roles remain separate.

With validation, ordinary DTO uniqueness supports plain String enum collection reference getters using runtime `serialName` equality and strict field-own defaults. Internal inline-enum collection references reuse their actual lexical enum declaration. See [String enum collection references](../../docs/dto-string-collection-reference-unique.md) for boundaries and migration.

Validated ordinary DTO uniqueness also supports existing finite List/Map getters with scalar/plain enum reference leaves. Ordinary inline fields gain strict own defaults for these source trees; unchecked native enum inline-default admission stays closed. See [collection reference leaves](../../docs/dto-collection-reference-leaves-unique.md) for actual source/getter authority and boundaries.

Validated ordinary allOf DTO uniqueness uses actual inherited getters and the original complete winning-field/required/default view. Existing aggregation, nullable and source/check identity rules remain. See [allOf DTO uniqueness](../../docs/dto-allof-unique.md) for boundaries, tuple approximation and migration.

With validation, admitted ordinary allOf default walks accept complete same-kind/format/nullable primitive override groups when earlier occurrences have no bounds. Original own annotations remain strict; outer values and omissions use the winning checks/default, without new emitted aggregation or default inheritance. See [allOf default overrides](../../docs/allof-default-overrides.md) for supported scalar and source roles.

Validation-on finite original native numeric/Boolean enum tuple references reuse existing scalar codecs/checks; see [the tuple contract](../../docs/tuple-native-enum-references.md). Distinct/mixed sources retain raw projection and the approximation warning. Unchecked/direct positional enums and general tuple/default graphs are separate.

Boolean nullable siblings on original complete recursive JSON cover graph refs/transparent named aliases are supported under the [edge contract](../../docs/nullable-recursive-cover-edges.md); carrier/wire stays JsonElement/JsonNull, while exported aliases/consumers retain existing nullable intent. General nullable recursion and constrained/inline covers stay separate.

OpenAPI3.1 literal false named components and pure aliases use the [uninhabited root contract](../../docs/false-component-roots.md). Non-null roots have no valid instances/JSON values; nullable aliases retain existing nearest intent. False fields/containers/HTTP/default/compositions and other source roles remain excluded.
