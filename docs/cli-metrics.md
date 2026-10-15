# CLI phase metrics

`generate`, `validate` and `generate --check` accept the bare opt-in `--metrics` flag. Repeating it or passing a value is a usage error. File/stdin input shares the same bounded-read metrics; no download phase/command exists. Default reports remain unchanged.

```bash
openapi-kt generate --input tests/fixtures/catalog.json \
  --output build/generated/catalog --package-name org.example.catalog \
  --metrics --diagnostic-format json
```

The stdout JSON report gains an optional `metrics` object:

```json
{
  "total_ns": 1000000,
  "phase_ns": {"read": 100000, "generate": 700000, "commit": 100000},
  "core_phase_ns": {"options": 10000, "parse": 100000, "analyze": 400000, "transform": 10000, "emit": 100000},
  "process_peak_rss_bytes": 16777216,
  "input_bytes": 256,
  "planned_files": 2,
  "planned_bytes": 1024
}
```

These illustrative numbers are not benchmark results. Durations use elapsed monotonic time, in integer nanoseconds. Zero is valid. Only entered phases appear:

| Phase | Boundary |
|---|---|
| `config` | Configuration open/limited read/parse/selection/settings/inventory validation; only in config mode |
| `name_lock_read` | Read-only previous lock metadata loading; only for selected locked generation/validation, including failure |
| `read` | Selected local file open/limited read/close or caller-owned stdin limited read without close, including failure |
| `generate` | Entire in-memory core generation: options validation, loading, analysis and emission together |
| `warning_policy` | Acceptance policy, only with `--deny-warnings`, including denial |
| `check` | Read-only output comparison, including difference/failure |
| `commit` | Output planning/write/transaction/no-op, including failure; not present for validate/check |

`total_ns` covers command execution from before configuration/previous-state/input open to the returned report. It excludes process startup, argument parsing and report formatting; external acquisition is not part of the CLI. `core_phase_ns` separately reports the entered core boundaries below. It is contained within `phase_ns.generate`; do not add these nested durations to coarse phases when calculating a total. `process_peak_rss_bytes`, when available, is a process-lifetime high-water mark, not memory attributable to one core phase or command. No allocation, declaration count, Kotlin compilation cost or speedup claim is made.

`input_bytes` counts bytes actually returned by the size-limited reader, possibly partial on read failure or limit+1 on oversized input. `planned_files` and `planned_bytes` count successful Kotlin core artifacts and UTF-8 source bytes. Name-lock metadata is excluded from these counts and from `input_bytes`; only its entered read phase is timed. They describe the generated plan, **not newly written files**: warning denial or output failure can leave these nonzero while report `files` is empty. Generation failure leaves planned counts zero.

Metrics are included on execution errors, warning denial, check differences and successes. Usage errors/help/version have no execution metrics. Exit codes, diagnostics, original warning visibility and ownership rules do not change. In text mode the usual stdout/stderr contract remains, with one additional stderr line beginning `metrics:`; phase fields follow the table order. JSON mode remains one stdout document and empty stderr.

Timing and RSS are CLI metadata, deliberately nondeterministic; no metric is included in core options, Kotlin sources, manifests, artifact hashes or sidecar files. Measured re-generation/check still preserves unchanged bytes and modification times. The default path does not read a clock. Use equivalent inputs/options/toolchains/cache conditions and repeated correctness-checked samples for any performance comparison; see [validation and performance](validation-and-performance.md).

## Core boundaries and process memory

| Core phase | Boundary |
|---|---|
| `options` | Generation-option and previous name-lock profile/format validation |
| `parse` | In-memory document loading, format selection and ordered JSON parsing |
| `analyze` | Component/operation analysis, reference resolution, initial naming, retained declarations and resource checks |
| `transform` | Optional hoisting/name-lock reconciliation, deduplication, merging and plain-class transformation |
| `emit` | Kotlin model/runtime/client emission, file sorting and final artifacts/name-lock encoding |

References and initial names are interleaved in the analyzer; there are no invented independent `resolve` or `name` timings. A failure closes its actual entered phase and omits later phases. A malformed document has `options` and `parse`; an analyzer failure also has `analyze`; an invalid previous lock is an `options` failure. `transform` appears even when all transformations are disabled, since the boundary is still entered. Text reports append `core_options_ns`, `core_parse_ns`, `core_analyze_ns`, `core_transform_ns`, `core_emit_ns` in this order, for entered phases only.

Core callers can use `GenerateWithPhaseObserver(request, observer)`. The synchronous caller-owned observer gets a phase and returns an optional completion callback. Each completion is called once, including on an ordinary generation error. Observers receive no source/plan objects and must not panic or mutate the request; separate concurrent generations need separate collectors or synchronization. `Generate`/`GenerateWithDiagnostics` use no observer, clock or OS calls. The [observer contract test](../core/phase_observer_test.go) checks identical artifacts/diagnostics and closure at failure boundaries.

On Darwin/Linux the CLI reads `getrusage(RUSAGE_SELF)` once at the end of enabled execution and normalizes Darwin bytes/Linux KiB to `process_peak_rss_bytes`. A native CLI process therefore includes startup, input, core and output memory used so far; an embedded caller using multiple executions sees a cumulative process high-water mark. It excludes child-process memory. It cannot assign memory to a spec/phase, infer current RSS, or measure a command-specific allocation delta. Other hosts and unavailable/nonpositive/overflow values omit the field. No RSS read occurs with metrics disabled. Usage/help/version never create an execution collector. The reported `total_ns` ends before the RSS sampling/report formatting overhead.

Existing machine consumers of `--metrics` should accept the additive optional fields. Default JSON reports, exit codes, diagnostics, generated files and ownership metadata are unchanged. See [validation and performance](validation-and-performance.md) for independent external RSS samples and repeated correctness-checked measurements; one high-water mark is not a distribution or a resource guarantee.

## Configuration aggregation

`--config ... --metrics` adds `config_bytes` to metrics (including zero when configuration cannot be opened). It counts actual limited-reader configuration bytes, at most the configuration limit+1; `input_bytes` excludes these and sums actual reads of **selected** source documents. The same input read for two specs counts twice. Unselected inputs are not read.

Repeated `name_lock_read`/`read`/`generate` durations are accumulated; warning policy and batch check/commit are measured once when entered. An early configuration error has only `config`; a later input/generation failure retains actual earlier phase/input counts. Empty reconcile has no invented source-read/generate phase. In text reports, `config_bytes` appears before the phase fields; direct-input telemetry stays unchanged.

Planned counts describe the **complete, unambiguous selected generated envelope**, not writes or retained-owner artifacts. Identical `(declared root, relative path)` candidates are counted once, so shared runtime helpers are not double-counted. Equal paths in distinct roots count separately. If a selected input/generation fails before the complete envelope, or selected candidates conflict at one root/path, planned counts are zero. Collision metrics do not change warning/output diagnostic ordering. Once a complete envelope is available, denial, a busy root or other output failure can still have nonzero planned counts and no written files. Metadata-only ownership adoption also counts its generated candidate sources even though it rewrites none. Empty reconciliation has zero planned counts regardless of how many old files it removes.

[Fake-clock/union tests](../internal/cli/config_metrics_test.go), actual native [aggregation](../internal/cli/config_metrics_process_test.go), [failures/text/partial/empty counts](../internal/cli/config_metrics_guards_process_test.go) and [Unix busy shared root](../internal/cli/config_metrics_unix_process_test.go) verify the config surface. Core subphase durations are also accumulated over selected specs, including an earlier successfully emitted spec before a later failure. Empty reconcile has no core subphase map. They do not provide per-spec reports, per-phase RSS/allocations or downstream Kotlin measurements.

[Native CLI tests](../internal/cli/metrics_process_test.go), [parser/fake-clock tests](../internal/cli/metrics_test.go) and [Unix busy-lock tests](../internal/cli/metrics_unix_process_test.go) cover actual phase boundaries, counts, default report equivalence, failures, warning policy, check/no-op and output preservation. These tests do not establish Windows/Linux runtime or performance/resource guarantees. [Native core metrics](../internal/cli/core_metrics_process_test.go) additionally cover the nested boundaries, failures, config aggregation and unchanged output.
