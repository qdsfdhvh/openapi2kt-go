# Model merge pattern and suffix exclusions

Opt-in [resolved model merging](merge-models.md) accepts three independent exclusion lists. A matching eligible root neither absorbs another root nor is absorbed. Existing exact-name matching remains unchanged.

| Core option | CLI flag (repeatable) | Config string array | Match target |
|---|---|---|---|
| `ExcludeMergeModels` | `--exclude-merge-model` | `exclude_merge_models` | Exact generated fully qualified root name |
| `ExcludeMergeModelPatterns` | `--exclude-merge-model-pattern` | `exclude_merge_model_patterns` | Entire generated fully qualified root name |
| `ExcludeMergeModelSuffixes` | `--exclude-merge-model-suffix` | `exclude_merge_model_suffixes` | Literal suffix of generated short root classifier |

```sh
openapi-kt generate --input api.json --output build/generated \
  --package-name org.example.api --model-name-suffix Dto \
  --merge-identical-models \
  --exclude-merge-model-pattern 'org\.example\.api\.models\.(Alpha|Zeta)Dto' \
  --exclude-merge-model-suffix ErrorDto
```

Matching is case-sensitive. Names include the effective model suffix and reflect the actual merge-time owner after hoisting/deduplication. Regex `AlphaDto` does not match `org.example.api.models.AlphaDto`; use the whole FQ name or an explicit `.*` prefix. Suffix `ErrorDto` matches `FetchErrorDto`, while `errordto` does not. Unknown valid rules match no root. Any of the three lists can protect a root; rule order does not change results. Excluded referenced classifiers retain distinct nominal identity in dependent shape comparison.

Exclusions control the merge pass. They do not undo prior operation deduplication/hoisting, prevent those earlier passes, invent nested-only merge candidates, or alter opaque-owner safety. The same existing ordinary DTO/String-enum merge eligibility applies. Disabling merging with a nonempty list is an option error. Core returns `E_OPTIONS` and no partial report; invalid direct CLI options are usage errors (exit 2); invalid effective configuration settings return `E_OPTIONS` (exit 1) before selected input/state reads. The full inventory is validated, including unselected specs.

## Portable pattern grammar and limits

Patterns are executed by the Go matcher over safe ASCII generated names. They are not emitted to Kotlin and do not scan schema/default/example payloads. The admitted grammar has common Go/Kotlin meaning on those names: printable ASCII literals, escaped regex punctuation, `.`, `^`, `$`, alternation, ordinary capturing groups, simple positive/negative character classes and ranges, greedy `?`/`*`/`+`, and repetitions with finite bounds up to 64 (an unbounded upper repetition is also allowed). Whole-name matching is explicit and independent of caller anchors.

Lookaround, backreferences, flags, noncapturing groups in user patterns, named groups, Unicode/property/shorthand classes, quoting escapes, POSIX/nested/intersection classes, lazy/possessive quantifiers, control characters and non-ASCII source are rejected. Escapes are limited to literal regex punctuation: `\\`, `\.`, `\^`, `\$`, `\|`, `\?`, `\*`, `\+`, `\(`, `\)`, `\[`, `\]`, `\{`, `\}`. Malformed patterns fail explicitly. This is a declared portable subset, not full Java Regex compatibility.

At most 64 patterns are accepted, each at most 1,024 source bytes and 256 parsed syntax nodes. Repetition bounds are at most 64. These limits bound validation/compilation work; matching uses Go's linear-time regexp engine. At most 64 suffixes are accepted, each 1–256 ASCII letters/digits/underscores. Suffixes may begin with a digit or consist of underscores because they match a trailing fragment, not a standalone Kotlin identifier. Existing schema depth/work/member/constructor caps are unchanged.

## Configuration precedence and migration

```toml
config_version = 1
[defaults]
merge_identical_models = true
exclude_merge_model_patterns = ['.*\.ErrorDto']
exclude_merge_model_suffixes = ['PrivateDto']
[[specs]]
name = 'catalog'
input = 'catalog.json'
output_dir = 'generated'
package_name = 'org.example.catalog'
exclude_merge_model_patterns = []
```

Each list obeys builtins < defaults < spec < explicitly provided CLI list. A spec `[]` resets only that list; an explicit CLI pattern list replaces the effective pattern list, while exact/suffix lists remain. Multi-spec CLI generation-option overrides still require a single selected spec.

No config, manifest or name-lock format version changes. Existing requests without new lists generate identical files and default reports. Enabling an exclusion intentionally preserves a classifier that merging would otherwise alias: generated identity, descriptors, equality and nested access may differ from a previously merged output. Review those call sites when changing rules; wire fields and retained validation stay the same. Name-lock occurrences remain computed before merge, with normal state preservation.

[Core tests](../core/merge_exclusions_test.go), [portable grammar tests](../internal/naming/merge_exclusions_test.go), [native precedence/preservation](../internal/cli/merge_exclusions_process_test.go), the [synthetic fixture](../tests/fixtures/http/merge-exclusions.json) and [Kotlin consumer](../samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergeExclusionContracts.kt) cover these rules. `bash scripts/test.sh feature merge-exclusions` generates data/plain profiles with combined hoist/dedup/name-lock/runtime options and four Json configurations, verifying protected descriptors/types, aliases, constraints and controlled HTTP calls. Cross-platform execution, general recursive merge equivalence and every option combination remain separate acceptance work.
