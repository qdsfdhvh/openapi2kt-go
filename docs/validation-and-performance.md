# Validation and performance

[Direct nested temporal defaults](nested-temporal-collection-defaults.md) share per-layer counts, original Pointer/nullable/default grammar and the existing source48/checked16/slots240/default32KiB limits. Temporal bounds/uniqueness remain excluded; no performance or combined-resource claim is added.

[String enum collection bounds](string-enum-collection-bounds.md) reuse builtin String length/pattern semantics only at ordinary DTO direct inline List/typed Map leaves. Original sources, open enum codecs, strict defaults, per-layer null/count checks and optimization identities remain. The feature route adds Data/Plain × checked/unbounded controls and an independent 240 shallow field probe; no caps, heap or performance claims change.

[Referenced builtin collection field defaults](referenced-collection-field-defaults.md) reuse original source/TypeRef/check authority and the same bounded annotation validator after forward linking. Their feature route covers independent source48/checked16/slots240/default32KiB probes and typed consumers; no performance or combined-resource claim is added.

The [finite nested builtin default profile](nested-builtin-collection-defaults.md) retains opt-in counts at every List/Map layer and eligible builtin leaf checks through constructor/copy/serde/Patch/HTTP/SSE. Source48, checked16 and slots240 remain; annotations separately bound whole-tree nodes and original/canonical JSON bytes. Its feature route tests independent width/depth/byte compilation, without claiming performance gains or combined maximum resources.

Correctness, input coverage, generation latency, memory and binary size are separate measurements. A fast rejection is not successful generation, and a passing scalar fixture does not establish full OpenAPI compatibility.

## Executable fixture coverage

The [fixture inventory](../tests/fixtures/README.md) has 154 named JSON inputs: 38 successful cases, 116 rejection cases and 50 complete Kotlin source snapshots. Both the core and real CLI execute every case. Resource boundaries and output transactions have additional tests.

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 -v ./...
python3 -m unittest discover -s scripts -p 'test_*.py'
bash scripts/test.sh models --offline
```

The Kotlin smoke compiles actual outputs from all successful cases in one invocation, with the serialization compiler plugin. The [enum, primitive-component, nullable-composition, const-field, object-composition and typeless/native enum-context, inline primitive-union, inline object-union, standalone root-union, typed hierarchy, scalar projection, named root kind/null and primitive normalization extensions](model-semantics.md) add runtime assertions against freshly generated serializers. No handwritten substitute models are used; ordinary P1 DTO round-trips remain outside this check.

## Opt-in local specification validation

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -trimpath -o build/openapi-kt ./cmd/openapi-kt
python3 scripts/perf/spec_inventory.py \
  --binary build/openapi-kt \
  --spec sample=/absolute/path/to/spec.json \
  --output local/spec-validation
```

Repeat `--spec label=path` for multiple files. The tool only reads explicitly selected local inputs and never downloads references. Reports, JSON derivatives and generated model subsets are restricted to ignored `local/`, `private/` or `.pi/` directories. Input hashes are checked again after inspection. Do not move raw reports into public fixtures or documentation.

The report distinguishes:

1. **Whole-document result**, including version, size and format rejection.
2. **Every observed schema location**, including parameters, request/response schemas, component definitions, composition branches, arrays and dictionaries. Examples/default/const payload contents are not recursively interpreted as schema slots.
3. **Deduplicated model-property probes**. Only descriptive schema metadata is removed for grouping. Names, declaration order and semantic values remain. Each distinct schema is wrapped in a required model property under a 3.1 envelope, with its transitive local component definitions. Every original occurrence points to its executed probe.
4. **Component probes** with original component definitions and reference dependencies. Only components accepted in this context are combined into a separately checked generated subset.
5. **Atomic fixture associations**, separated into positive coverage, rejection-only coverage and missing fixtures. Syntax failures and unrelated earlier errors do not count as schema feature coverage. Associations are not proof of every possible combination.
6. **Operation/wire feature inventory**, explicitly marked `deferred-not-wire-tested` while HTTP generation remains unimplemented. Parameters, bodies, response media/statuses and SSE cannot be declared tested merely because non-empty `paths` is rejected.

A model-property probe is **not** request/response presence validation, HTTP wire validation or whole-document compatibility. For example, successful primitive probes do not make an enclosing union supported. A supported component subset is a derivative workload, not the original complete specification.

For `.yaml`/`.yml` inputs, an already-installed PyYAML can perform an optional local conversion for JSON model probes. The report keeps the native CLI rejection and converted-JSON result separate. This is an audit convenience, not a production YAML loader or a guarantee about all YAML constructs. JSON parse failures do not silently fall back to YAML.

## Benchmark commands

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go build -trimpath -o build/openapi-kt ./cmd/openapi-kt
python3 scripts/perf/benchmark.py \
  --binary build/openapi-kt \
  --models 100 1000 8000 --fields 16 --seed 17 \
  --iterations 7 --warmup 2 \
  --output local/benchmarks
```

Add `--input /path/to/an/accepted-model-document.json` to benchmark an explicitly accepted local input. Use `--models` with no counts when measuring only explicit inputs. Rejected inputs abort the benchmark instead of producing a misleading speed result.

Synthetic workloads contain actual scalar fields, required/optional properties, nullable list items, nested objects and cyclic component references. They are not tiny models hidden inside megabytes of ignored descriptions. Generation is deterministic for the recorded dimensions and seed.

### Measurement boundaries

- **Validate:** process startup + input read + parsing/analysis/emission, without output commit. This is not a pure-core or parse-only measurement.
- **Clean generation:** a new temporary output directory for each sample.
- **No-op generation:** existing identical output, including ownership and hash checks.
- **Check:** read-only comparison against existing output.

Each command must exit successfully and report the expected model count. Every generated file is checked against the ownership manifest; complete source fingerprints must agree across runs. No-op/check must preserve bytes and mtimes. Verification and temporary-directory cleanup occur **outside** the timed interval.

Runs use warm OS caches without cache flushing; “clean” does not mean cold disk. Reports retain every sample plus median, nearest-rank p95, minimum and maximum. With seven samples, p95 is simply the maximum and should not be overinterpreted. Run builds, schema audits and Kotlin compilation **before**, not concurrently with, timing.

Peak RSS is measured in one additional successful `/usr/bin/time` invocation per mode on macOS/Linux, normalized to bytes. It is not a distribution. Python's memory and correctness-check time are excluded from child RSS. Unsupported platforms report no RSS value rather than inventing one.

### Binary size

Reports record the exact executable byte size and SHA-256. When the host's `strip` tool is available, the harness strips a **copy**, records its flags/size/hash and verifies actual generation with that copy. It never modifies the supplied executable. A stripped-copy measurement is not a published artifact or a change to the Go compiler/linker flags.

Record host/CPU, Go/Python versions, binary version/hash, build profile, input hash/size, component count, output source bytes/file count and workload settings with every baseline. Do not compare machines or claim speedups without the same correctness-checked workload and measurement boundary.

## Bounded Kotlin emission

Model files are rendered concurrently after analysis, naming and optional model transformations finish. Each worker reads the finalized plan and owns its source buffer and result slots. File order and bytes are independent of scheduling; diagnostics retain their original analysis order. Output transactions still begin only after all artifacts have been generated.

Emission uses at most eight workers, capped by `GOMAXPROCS`, with roughly one worker per 32 top-level declarations. Batches below 64 declarations and `GOMAXPROCS=1` use the serial path. This is an internal scheduling policy, not a new CLI option. Parsing, shared schema analysis and filesystem transactions have separate costs, so an emission speedup does not imply the same end-to-end speedup.

The standalone emitter benchmark excludes parsing, analysis and filesystem IO. Its deterministic workload has 16 scalar/list/reference properties and one nested DTO per model, with no random seed or external input. Each iteration checks the complete ordered file contents against serial emission outside the timed interval:

```bash
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./internal/kotlin \
  -run '^$' -bench '^BenchmarkEmit$' -benchmem -benchtime=300ms -count=7
```

Use the CLI benchmark above for total generation latency and `--metrics` for parse/analyze/emit/commit attribution. Neither measurement establishes downstream Kotlin compilation improvements.

### Go emission measurements

One local comparison on Apple M2 Pro (10 logical CPUs), macOS 27.2 arm64, Go 1.27.2, default compiler optimization and `GOMAXPROCS=10` produced the following emitter medians across seven benchmark repetitions (`-benchtime=300ms`). The range is the minimum/maximum repetition mean, not a request-latency percentile. Fixed worker counts isolate scheduling; automatic emission uses three workers for 100 declarations and eight for the larger cases.

| Models | Serial median (range) | Eight-worker median (range) | Serial / parallel median |
|---:|---:|---:|---:|
| 100 | 1.085 ms (0.917–1.152) | 0.737 ms (0.717–0.781) | 1.47× |
| 1,000 | 8.460 ms (7.871–9.650) | 3.494 ms (3.023–3.731) | 2.42× |
| 4,000 | 32.565 ms (32.040–35.956) | 11.097 ms (10.619–14.259) | 2.93× |

For 4,000 models the median allocated bytes were 36,325,424 serial and 36,327,034 parallel per operation. Parallel emission reduces elapsed render time, not total allocations.

### Go pipeline measurements

The pipeline also avoids building borrowed-leaf verification indexes when their recorded use sets are empty, reuses provisional ownership indexes when no hierarchy changes ownership, and skips raw-reference linking when no raw targets exist. Model/shape buffers reserve bounded capacity and reuse parent JSON Pointer strings. Small JSON objects use independent linear lookup snapshots; objects with more than eight members retain hash indexes. Duplicate-key, source-identity and resource-limit checks are preserved.

Output planning and locked application reuse directory checks within each pass. Private staged files are written with bounded concurrency before serial destination mutation. The [output ownership contract](output-ownership.md) describes cache lifetime, lock-time revalidation and rollback boundaries.

A native CLI comparison on the same host/toolchain used the Python synthetic workload (seed 17, 16 properties, cyclic references and nested objects), matching `go build -trimpath` binaries, warm OS caches, two warmups and seven samples per mode. The three variants were the original serial pipeline, parallel emission alone, and the optimized pipeline. Their execution order rotated per sample; all used `--metrics`. Counts, manifests, complete source fingerprints and no-op/check mtimes were verified outside timing, with identical source fingerprints across all variants. These CLI workloads differ from the emitter-only workload above.

| Models | CLI boundary | Original median / p95 | Parallel emission only median / p95 | Optimized pipeline median / p95 |
|---:|---|---:|---:|---:|
| 100 | Validate (no output commit) | 19.56 / 21.29 ms | 18.32 / 33.83 ms | 13.70 / 17.63 ms |
| 100 | Clean generation | 43.91 / 57.11 ms | 42.98 / 45.12 ms | 32.79 / 37.15 ms |
| 100 | No-op generation | 32.80 / 40.68 ms | 34.33 / 46.67 ms | 23.11 / 29.78 ms |
| 100 | Check | 24.76 / 26.13 ms | 23.86 / 28.05 ms | 18.09 / 21.77 ms |
| 1,000 | Validate (no output commit) | 112.44 / 118.14 ms | 102.61 / 123.15 ms | 70.44 / 74.42 ms |
| 1,000 | Clean generation | 357.32 / 499.35 ms | 354.49 / 389.70 ms | 252.24 / 301.58 ms |
| 1,000 | No-op generation | 195.93 / 237.01 ms | 188.98 / 206.70 ms | 126.85 / 139.75 ms |
| 1,000 | Check | 157.29 / 217.70 ms | 150.29 / 210.08 ms | 101.43 / 136.04 ms |
| 4,000 | Validate (no output commit) | 448.89 / 512.53 ms | 424.13 / 478.38 ms | 238.43 / 341.87 ms |
| 4,000 | Clean generation | 2,017.07 / 2,152.15 ms | 1,672.71 / 1,975.74 ms | 1,051.55 / 1,296.23 ms |
| 4,000 | No-op generation | 909.18 / 1,099.89 ms | 911.58 / 1,120.50 ms | 614.55 / 848.56 ms |
| 4,000 | Check | 814.94 / 2,409.59 ms | 684.31 / 1,534.17 ms | 450.64 / 1,950.64 ms |

With seven samples p95 is the maximum. For 4,000 models, analysis in validate fell from 317.39 ms with parallel emission alone to 156.00 ms with the optimized pipeline. Median process-lifetime peak RSS in those validate samples fell from 193.30 to 146.28 MiB. A separate in-process run of the same input measured median total allocated bytes per generation at 302.39 versus 214.54 MB (decimal), across seven samples after two warmups; this is allocation volume, not peak or retained memory. All in-process outputs matched the CLI source fingerprint.

The optimized pipeline reduced the 4,000-model validate and clean-generation medians by about 44% and 37% relative to parallel emission alone. Filesystem timings vary substantially: check improved at the median but had a worse maximum sample than emission alone. These observations do not establish uniformly better tail latency, cross-machine speedups or faster Kotlin compilation. Retain raw samples and compare equivalent runs rather than comparing these absolute values with a different baseline session.

## Go build baseline

The [Go implementation](go-port.md#go-build-baseline) records a correctness-checked compiler-cache cold/warm build and binary-size baseline. It does not establish generator latency or peak RSS. New end-to-end Go measurements use the commands above; no Go-versus-Rust speedup is established.

## Historical Rust generation baseline

The observations below belong to the retired Rust backend, not the current Go implementation. They are retained synthetic-data evidence only; do not compare them with Go build times or present them as current generator performance.

One local run on **Apple M2 Pro / arm64 / Darwin 27.2.0**, Rust/Cargo **1.97.1**, default Cargo release profile, seed 17 and 16 properties per component. Two warmups and seven measured samples per mode; timings below are end-to-end CLI medians in milliseconds, not pure-core timings.

| Components / Kotlin files | Input bytes | Validate | Clean generation | No-op generation | Check | Clean p95 | Clean peak RSS |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 100 | 95,246 | 7.35 ms | 25.21 ms | 16.68 ms | 12.39 ms | 67.39 ms | 6.62 MiB |
| 1,000 | 950,926 | 36.89 ms | 231.53 ms | 137.94 ms | 92.06 ms | 259.42 ms | 38.11 MiB |
| 8,000 | 7,607,526 | 302.37 ms | 1,670.32 ms | 1,165.53 ms | 730.46 ms | 1,968.39 ms | 282.27 MiB |

The largest workload produced **18,224,472 source bytes in 8,000 files**. Every measured run passed count/hash/determinism checks; no-op and check preserved mtimes. RSS is one additional run per mode. The 100-model clean samples had an outlier, illustrating why raw distributions matter.

Binary size was **1,367,616 bytes** (about 1.30 MiB). A separately verified `strip -S -x` copy was **1,141,184 bytes** (about 1.09 MiB); the original executable was unchanged. No LTO, panic-strategy or release-profile tuning was added for this measurement.

[The synthetic-only baseline JSON](benchmarks/baseline-macos-arm64.json) includes raw timing samples, input/output hashes, source fingerprint, toolchain and executable hashes. These are observations, not performance guarantees or cross-platform comparisons. Local specification-derived workloads are deliberately excluded from the published baseline.

## Interpretation limits

Current generation is model-only and accepts at most 8 MiB of input with OpenAPI 3.0.x/3.1.x. Version 3.2, larger documents, operations and advanced schema semantics can fail before emission. Their rejection time is not a generation benchmark. Public reproducible baselines should use synthetic data; raw local reports and generated models remain local.
