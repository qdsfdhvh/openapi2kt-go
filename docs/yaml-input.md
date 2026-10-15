# YAML document input

OpenAPI **3.0.x and 3.1.x** can be supplied as one YAML document through the same
pure-memory core, native file/stdin commands and local configuration as JSON.
The adapter uses the vendored `go.yaml.in/yaml/v3` v3.0.4 parser and constructs the
existing ordered JSON-compatible `document.Value` tree. Analysis, Kotlin naming,
emission, warnings and output transactions share the JSON implementation.
Equivalent accepted JSON/YAML trees, with the same member/list order and number
tokens, produce identical Kotlin files, APIs, wire behavior and semantic
Pointers. The adapter does not extend the supported OpenAPI/JSON Schema subset.

```sh
openapi-kt generate --input openapi.yaml --output generated --package-name org.example.api
openapi-kt validate --input openapi.yml
openapi-kt validate --input - --input-format yaml
```

`core.Auto` / `--input-format auto` selects JSON for `.json`, YAML for `.yaml` or
`.yml` (case-insensitive), and JSON for an extensionless source, including stdin.
A leading dot alone is not an extension. Other extensions require explicit
`json` or `yaml`. Explicit JSON never retries a parse failure as YAML. Explicit
YAML also accepts JSON syntax; YAML's own scalar/key rules still apply.
Configured `input_format = 'yaml'` works in defaults/specs and a selected CLI
override; configured input remains a local file, never a stdin marker.

## Accepted and rejected YAML

This is a bounded JSON-compatible YAML profile, not every YAML representation.
Rules apply to the entire document, including ignored annotations/examples and
unused anchors. Quotes or `!!str` express a string when implicit scalar typing
would give another kind.

| Construct | Policy |
|---|---|
| Block/flow maps and sequences; quoted, literal and folded strings; comments | Accepted; parser-defined string folding/escapes and declaration order retained. Comments are not part of the generated tree. UTF-8 BOM is accepted. |
| Mapping keys | Must be scalar strings; no coercion of numeric/boolean/null/date keys, complex keys or aliases used as keys. Duplicate decoded keys are errors, including quoted/escaped equivalents and example payloads. |
| Numbers | Only finite JSON number syntax: optional minus, decimal integer, optional fraction/exponent. Exact token retained without a float64 conversion in the tree. Hex/octal, leading zeros, underscores, plus-prefixed numbers, `.5`, `1.`, NaN/Inf and overflowing plain JSON numbers are rejected. Quoted strings stay strings. |
| Booleans/null | Resolved true/false spellings and null/`~`/empty values become JSON booleans/null. `yes`/`no`/`on`/`off` remain strings; `!!bool yes` is rejected. Quote `null` in a 3.1 type list when the intended value is the string `"null"`. |
| Implicit dates/timestamps | Scalar **values** remain original strings; no `time.Time`, timezone conversion or date validation. Explicit `!!timestamp` is rejected. Quote date-looking keys. |
| Tags | Only JSON-compatible `!!map`, `!!seq`, `!!str`, `!!int`, `!!float`, `!!bool`, `!!null` with compatible node/value kinds. Custom tags, including verbatim/local URIs that only normalize to a builtin spelling (`!<!!str>`, `!<%21%21map>`, `!%21seq`), are rejected using original node-property identity. Standard `!!` shorthand and canonical `tag:yaml.org,2002:` URIs (including valid percent encoding) remain accepted subject to the same kind/value checks. Binary, sets, ordered maps, pairs and explicit timestamps are rejected. Bare `!` follows parser implicit resolution. `%TAG` directives are rejected before parsing to prevent upstream URI-prefix allocation amplification. The conservative guard rejects any logical source line beginning `%TAG` (including BOM-prefixed lines and quoted continuations); indent the text or escape `%` when it is string data. Other directives have only parser syntax effects and do not select a different product scalar profile. |
| Anchors/aliases | Unique anchor declarations and defined finite aliases accepted in values. Expand in occurrence order; every occurrence gets the same tree/value semantics as explicit JSON. Rebinding, undefined aliases and cycles are rejected. Anchors are unrelated to OpenAPI `$ref`; external refs stay unsupported. |
| Merge keys | `<<`/`!!merge` rejected, including override/merge sequences. Write members explicitly. Quoted `"<<"` is an ordinary string key. |
| Streams | Exactly one document. Optional document start/end markers accepted; a second document (even empty), trailing YAML or empty/comment-only input fails. |

## Resource and diagnostic boundary

- Raw input is at most **8 MiB**, including whitespace, comments and metadata.
  The CLI reads at most the limit plus one detection byte. Invalid UTF-8 and
  escaped surrogate/out-of-range Unicode scalars fail; UTF-16 input is unsupported.
- Source and expanded JSON-compatible trees each allow **127 simultaneously open
  containers**, matching the existing JSON limit. Each alias target is checked at
  its occurrence depth. Alias paths independently allow at most **127 hops**.
- YAML source traversal and expanded traversal each allow **100,000 node visits**,
  including mapping keys; alias expansion counts the alias and its target.
  Expansion also allows at most **8 MiB** of compact JSON bytes as serialized by
  the shared tree (including escaped strings, keys and punctuation). A short
  source can exceed these budgets through aliases or string escaping.
- These node/tree checks follow the upstream bounded syntax-to-Node parse. An
  independent preflight rejects `%TAG` lines before the parser can expand URI
  prefixes. It covers LF/CR/CRLF/NEL/Unicode line/paragraph separators and
  column-zero BOM. The vendored parser separately limits flow/indent nesting to 10,000; its temporary
  Node/comment allocations are bounded by input size and that syntax limit, not
  by a hard RSS guarantee or the later 100,000-visit acceptance check. The adapter
  never uses upstream typed decoding or its implicit alias/merge expansion.
- Existing schema depth/work, collection/constructor/JVM-slot limits stay intact.
  JSON uses its existing parser, limits, artifacts and warning order.

`E_INPUT_LIMIT` identifies oversized raw input. `E_YAML_INVALID` covers invalid
syntax/UTF-8, duplicates/rebinding, undefined aliases and multiple documents;
`E_YAML_UNSUPPORTED` identifies unsupported keys/tags/scalar forms/merge keys;
`E_YAML_ALIAS_CYCLE` identifies cycles; `E_YAML_LIMIT` identifies source/expansion
budgets. These are generation/input exit **1** errors and return no artifacts.
Node errors include available line/column coordinates; raw parser errors are
replaced by safe messages. They never echo source names or payloads. Non-object
roots retain `E_DOCUMENT_INVALID` with the empty root Pointer. Subsequent analysis
uses the same semantic JSON Pointers and diagnostics as equivalent JSON; an
alias's semantic Pointer describes the expanded occurrence. There is no general
source-span map or alias provenance report.

File/stdin/config parse or warning-policy failures preserve generated and
handwritten output. Check remains read-only; configuration batches generate all
selected inputs before committing. No filesystem, environment, subprocess or
network access is added to core. No remote YAML acquisition or `$ref` fetch is
provided.

## Compatibility and verification

YAML acceptance is additive; existing JSON output, warnings, CLI exit codes,
configuration version and manifest/name-lock formats are unchanged. Existing
scripts that intentionally depended on YAML rejection must use explicit JSON or
validate against this profile. Historical JSON alpha artifacts do not acquire
YAML support; a new release needs its own candidate/platform/package evidence.
Full YAML, merge semantics, non-JSON tags/keys, general source spans and all
cross-platform runtime combinations remain outside this profile.

The independent [JSON](../tests/fixtures/yaml/records.json) /
[YAML](../tests/fixtures/yaml/records.yaml) synthetic fixture covers models,
anchors, open enums, defaults, nullable fields, HTTP and request Patch. Evidence
entry points are [ordered-tree/guards/fuzz](../internal/document/yaml_test.go),
[core parity](../core/yaml_test.go),
[native file/stdin/config/failure preservation](../internal/cli/yaml_process_test.go)
and `bash scripts/test.sh feature yaml-document` for fresh JSON/YAML byte parity,
Data/Plain × validation on/off compilation and four Json setting combinations
with serde/Patch and controlled-engine GET/query/PATCH/response assertions.
`bash scripts/test.sh models` separately checks the existing fresh model suite.
Actual executed checks and hosts belong in candidate/PR evidence, not inferred
from test availability. See [dependency notices](../THIRD-PARTY-NOTICES.md).
