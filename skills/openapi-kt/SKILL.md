---
name: openapi-kt
description: Use when validating OpenAPI JSON or bounded YAML from local files or stdin with openapi-kt, generating Kotlin sources, adopting direct locked output into a named owner, renaming configured output owners, explicitly moving complete owned output roots, checking generated-source drift, or integrating the generated models and Ktor clients.
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
| Adopt a direct locked owner | [Adopt a direct locked owner](references/commands.md#adopt-a-direct-locked-owner) | Verify complete same-root v3 inventory/raw lock and exactly one configured owner; review hashes/classifier/descriptor report and confirm subsequent named config `--check` |
| Rename a configured output owner | [Rename a configured owner](references/commands.md#rename-a-configured-owner) | Verify same-root v2/v4 ownership and unchanged Kotlin/lock semantics, review `owner_renames`, then confirm ordinary generation under the new name with `--check` |
| Move a complete owned output root | [Move an output root](references/commands.md#move-an-output-root) | Verify explicit source/target and complete original inventory, review preserved bytes/hashes/state, then confirm ordinary target generation with `--check` |
| Check local or CI drift | [Check drift](references/commands.md#check-drift) | Report consistency, differences or safety errors; no writes |
| Inspect nested String enum field defaults | [Nested String enum defaults](references/kotlin.md#nested-string-enum-collection-defaults) | Check the finite direct inline outer-field scope, preserve source diagnostics and presence, and report JVM evidence separately from PREPARE_ONLY |
| 检查嵌套 numeric/Boolean enum 字段默认值 | [Nested native enum defaults](references/kotlin.md#nested-native-enum-collection-defaults) | 核对 direct inline outer-own 角色、原 exact membership/presence/checks 和 native warning；分开报告 source 与实际 JVM 证据 |
| Inspect referenced String enum field defaults | [Referenced String enum defaults](references/kotlin.md#referenced-string-enum-collection-defaults) | Check the direct `$ref` field own-default role, strict original vocabulary, and report JVM evidence separately from PREPARE_ONLY |
| Inspect referenced String enum null-member defaults | [Referenced null-member defaults](references/kotlin.md#referenced-string-enum-null-member-defaults) | Check original explicit nullable intent, actual narrowed view and null versus String `"null"` |
| Inspect referenced String enum scalar defaults | [Referenced String scalar defaults](references/kotlin.md#referenced-string-enum-scalar-defaults) | Check field-own annotation, actual lexical classifier, strict vocabulary and open unknown Strings |
| Inspect referenced native enum scalar defaults | [Referenced native scalar defaults](references/kotlin.md#referenced-native-enum-scalar-defaults) | Check field-own annotation, existing reference admission and strict vocabulary before scalar mapping |
| 检查 referenced ordinary DTO object 字段默认值 | [Referenced object defaults](references/kotlin.md#referenced-ordinary-dto-object-field-defaults) | 核对 ordinary direct local `$ref` field own `default`、实际 nominal 类型/helper 绑定、nearest nullable 与 target-root 排除；JVM 证据与 PREPARE_ONLY 分开 |
| 检查 inline primitive union 字段默认值 | [Union field defaults](references/kotlin.md#inline-primitive-union-field-defaults) | 核对 native primitive-union 声明、token 保留与 runtime selector wrapper、nullable/required/Patch；pin/object/ref/format 分支仍排除 |
| 检查 referenced ordinary DTO 元素 collection 字段默认值 | [Referenced object collection defaults](references/kotlin.md#referenced-ordinary-dto-elements-in-collection-field-defaults) | 核对 inline `$ref` DTO 元素与 named collection `$ref` DTO 元素、helper 绑定与实际类型；element own/target-root default 仍排除 |
| 检查 inline List/Map of ordinary DTO object 字段默认值 | [Object collection defaults](references/kotlin.md#direct-inline-listtyped-map-of-ordinary-dto-object-field-defaults) | 核对 ordinary direct inline `array`/`Map` 字段 own `default`、元素对象递归、逐层 nullable 与 helper wrapper 绑定；element own default 与 referenced 元素仍排除 |
| 检查 inline ordinary DTO object 字段默认值 | [Nested object defaults](references/kotlin.md#direct-inline-ordinary-dto-object-field-defaults) | 核对 ordinary direct inline plain object own `default`、direct local `$ref`-valued properties、required/Patch、逐层 nullable 与 original Pointer；JVM 证据须与 PREPARE_ONLY 分开报告 |
| Inspect referenced native enum field defaults | [Referenced native enum defaults](references/kotlin.md#referenced-native-enum-collection-defaults) | Check scalar-wire vocabularies, exact default membership and validation bounds, and report JVM evidence separately from PREPARE_ONLY |
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
- `validate` checks this generator's supported subset, not complete OpenAPI or JSON Schema validity. Review approximation warnings even on success, and preserve unsupported semantics in the original spec.
- Local configuration uses bounded TOML 1.1 syntax for the existing closed `config_version`/`defaults`/`specs` schema. Equivalent quoted/dotted keys, multiline strings, inline tables and standard integer spellings preserve path/precedence/ownership behavior; types remain schema-defined. Follow [Inputs and options](references/commands.md#inputs-and-options) for config budgets and rejected metadata.
- Direct output has one owner; configuration roots retain named-owner membership. Explicit owner rename preserves all peers and shared memberships, including peers absent from config, and uses a metadata-only transaction in the same root. Explicit complete root move preserves v1–v4 bytes/attributes and all original ownership; it requires a concrete source and target, full same-root config inventory and unchanged options. Partial/disjoint inventory moves remain unsupported; explicit same-root locked-owner adoption supports complete direct v3→one named v4 owner with exact Kotlin/raw-lock preservation. Keep the manifest and any owned name-lock metadata with generated files, and custom code outside generated files. Do not remove manifests, locks, recovery backups or the output root to bypass a safety error.

Report actual results and unverified items. Match verification to the task: CLI work needs no blanket Kotlin smoke run; integration uses the consumer project's relevant compilation/tests and dependency policy. Existing JVM smoke evidence does not verify a new Android/KMP target.
