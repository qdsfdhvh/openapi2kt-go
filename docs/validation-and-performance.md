# Validation and performance

[Direct nested temporal defaults](nested-temporal-collection-defaults.md) share per-layer counts, original Pointer/nullable/default grammar and the existing source48/checked16/slots240/default32KiB limits. Temporal bounds/uniqueness remain excluded; no performance or combined-resource claim is added.

[String enum collection bounds](string-enum-collection-bounds.md) reuse builtin String length/pattern semantics only at ordinary DTO direct inline List/typed Map leaves. Original sources, open enum codecs, strict defaults, per-layer null/count checks and optimization identities remain. The feature route adds Data/Plain × checked/unbounded controls and an independent 240 shallow field probe; no caps, heap or performance claims change.

[Referenced builtin collection field defaults](referenced-collection-field-defaults.md) reuse original source/TypeRef/check authority and the same bounded annotation validator after forward linking. Their feature route covers independent source48/checked16/slots240/default32KiB probes and typed consumers; no performance or combined-resource claim is added.

The [finite nested builtin default profile](nested-builtin-collection-defaults.md) retains opt-in counts at every List/Map layer and eligible builtin leaf checks through constructor/copy/serde/Patch/HTTP/SSE. Source48, checked16 and slots240 remain; annotations separately bound whole-tree nodes and original/canonical JSON bytes. Its feature route tests independent width/depth/byte compilation, without claiming performance gains or combined maximum resources.

Correctness, input coverage, generation latency, memory and binary size are separate measurements. A fast rejection is not successful generation, and a passing scalar fixture does not establish full OpenAPI compatibility.

The [discarded const validation measurements](discarded-const-validation.md) cover a profiled temporary-initializer bottleneck with repeated shared, distinct and overridden accepted literals. Every before/after sample checks exact Kotlin and report equality. The validation-only path preserves source grammar, diagnostics, actual retained initializers and existing caps; native latency and core allocated bytes do not establish Kotlin compilation, artifact-size or RSS savings.

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

## Primitive union merge resource sample

A local macOS27.2 ARM64 / Go1.27.2 `-trimpath` comparison used Kotlin2.4.10, serialization1.11.0, Gradle9.5.1 and JDK21 with JVM17 output. Both synthetic inputs have 120 roots named R000–R119. The repeated input uses `oneOf` String/integer/number/Boolean in that order, `nullable:false` on every root. The unique control enumerates ordered permutations of those kinds with lengths2–4, each with nullable false then true: 120 distinct emitted signatures. Package is `org.example.measured`, suffix Dto, default Data style and validation off; only the explicit merge flag differs.

[Raw samples and exact workload/toolchain definitions](benchmarks/primitive-union-merge-macos-arm64.json) retain input/binary/source/JAR hashes. CLI clean generation includes process startup, input/core and commit to a fresh owned output: two warmups, seven measured samples per mode, alternating baseline/merge order. Every manifest path/hash and complete source inventory is verified outside timing, and unique-control sources match byte-for-byte. Kotlin uses the HTTP smoke Gradle template with generated sources only, offline dependencies, no build cache, rerun jar tasks, one worker and in-process compilation: one warmup and three measured whole-Gradle runs per mode, also rotating order. JAR bytes/class counts are stable and all observed JAR hashes are retained. Builds are sequential; OS caches remain warm.

| Workload / cost | Merge off | Merge on |
|---|---:|---:|
| Repeated: generated Kotlin bytes / files | 621,120 / 120 | 57,179 / 120 |
| Repeated: clean CLI median | 23.86 ms | 23.11 ms |
| Repeated: Kotlin jar build median | 10.81 s | 5.75 s |
| Repeated: JAR bytes / class files | 2,330,932 / 1,800 | 88,254 / 134 |
| Unique: generated Kotlin bytes / files | 600,240 / 120 | 600,240 / 120 |
| Unique: clean CLI median | 25.43 ms | 25.90 ms |
| Unique: Kotlin jar build median | 10.61 s | 10.72 s |
| Unique: JAR bytes / class files | 2,119,551 / 1,632 | 2,119,551 / 1,632 |

This measures the two documented workloads and selected dependency/host combination. It shows lower repeated-source and downstream artifact costs; the unique control has unchanged sources/artifacts and slightly higher observed elapsed time with merge. Small CLI differences and three Kotlin samples do not establish a general speedup or latency distribution. Kotlin timing includes single-use Gradle startup; compiler-only time and compiler/daemon peak memory were not measured. These are not width×depth Cartesian, arbitrary union/hierarchy, minimum-toolchain or cross-platform resource bounds. Functional wire/alias evidence belongs to the [separate merge contract](merge-models.md#primitive-json-kind-union-roots).

[Dependency-combination evidence](dependency-combinations.md) separately records complete JVM models/HTTP source and runtime checks under selected lower compiler/runtime versions, with actual resolved versions and failed UUID/SSE combinations. These are compatibility results, not generation/compilation performance measurements or general minimum-version guarantees.

[Repeated deep collection checks](wide-deep-validation.md) move oversized inline traversal bodies into shared private typed methods, retaining actual type/check/null/Patch identity, names, state-free behavior and public API/wire/lock contracts. The resource route checks 200×List-depth16 and 240×depth12 in Data/Plain×all8 optimization combinations, four Json settings, factories/copy/HTTP/SSE under original heap/source limits with complete inventory partitioning. Independent caps are not a full Cartesian promise: JVM generic-signature UTF8 feasibility, other shapes/toolchains/platforms and broader D08 remain open.

[Certain JVM constructor signature overflow](jvm-signature-limits.md) reports E_MODEL_LIMIT at the original retained-model source before artifacts when fixed known List/Map/boxed-builtin signature bytes alone exceed JVM65535-byte UTF8 capacity. Unknown/alias/wildcard bytes are omitted, so this is a one-sided proof rather than complete backend feasibility. Proven compiled200×16/240×12, public types/wire/state and existing caps remain; previously emitted but unrepresentable inputs now fail safely. Wider signature/alias/name/platform cases stay open.

[UUID and temporal form fields](standard-form-fields.md) extend A05/A08/D07 to direct required/optional Uuid/LocalDate/Instant properties. Existing canonical UUID/ISO value text is Ktor form-encoded; optional keys and body absence remain independent. Shared HTTP/SSE encoding, UUID opt-in checks, original AST/retained fields and source/checked/JVM240 budgets remain. References/defaults/schema-null/formatted enums/native overrides and temporal bounds retain their boundaries. Accepted API/wire/state formats are stable; wider roles/platforms remain open.

[Outer allOf UUID defaults](allof-uuid-defaults.md) extend D03/D05 for complete original ordinary string/uuid contribution groups under validation. Retained length/pattern checks inspect canonical Uuid text, while original own/winning/outer annotations, actual constructor/codec, required/Patch, source Pointers and shared budgets remain. Invalid values fail at original annotations; no schema copying, new codec or persisted identity is introduced. Wider references/enum/formats/defaults/aggregation/platforms stay separate.

[Known inline List variance limits](jvm-list-variance-limits.md) tighten the one-sided JVM constructor proof only for complete direct List chains ending builtin6, including optional roots. Known nested List covariance rejects impossible204×16 while preserving freshly compiled203×16 Data/Plain and earlier200×16/240×12. Original Pointers, no artifacts/failure preservation, public types/wire and all existing caps remain. Alias/Map/nullable-element/mixed/copy/name exactness and wider platforms/general D08 remain open.

[Retained Data copy signature limits](jvm-copy-signature-limits.md) add a post-transform lower-bound check using known parameter syntax/List variance plus the ASCII package return prefix. Final Plain classes stay admitted without a Data copy API; impossible Data signatures fail at original Pointers with no artifacts/old-output changes. Private Plan diagnostics remain outside declaration/lock identity and source permissions. Existing types/wire/caps/state formats hold; exact names/aliases/other metadata/platforms and wider D08 stay open.

[Named DTO bridge intersections](bridge-intersections.md) extend D01/D02/F04/F07 to already-bound root bridges through finite internal chains ending at original direct named non-null ordinary DTOs. The allOf planner reuses the real catalog/node/field Pointers and full cached height, preserving original constructors, nested copies, defaults/checks, required/Patch, recursion and approximation warnings. Nullable/composed/internal lexical/union/hierarchy/count/raw roles remain separate; source/work/checked/JVM240 caps, identity and failure preservation hold. Existing API/wire/state formats are stable; broader graph/platform completion stays open.

[Nullable named DTO bridge intersections](nullable-bridge-intersections.md) extend D01/D02/F04/F07 to intrinsic target and nearest bridge/member null intent in the existing actual named ordinary allOf role. Early discovery follows original internal ref edges without admission, preserves every named/internal node's work/depth before and after binding, and final source/catalog/constructor checks remain authoritative. Nullable codecs/AND intersection/required/Patch, original fields/AST/owners, caps and failure preservation retain their contracts. Other endpoint/shape/platform roles remain separate; accepted non-null output/state formats are stable.

[Actual internal DTO intersections](internal-dto-intersections.md) extend D01/D02/F04/F07 to direct/internal-chain/root-bridge allOf operands with unique actual ordinary lexical constructors. Resolved carriers keep original nodes/Pointers and real named catalogs without creating internal component identities; existing final owner proof rejects copies/scratch/missing owners. Shared document capture caps16,384work/8MiB charge overlapping targets while repeated targets reuse captures; prior source/composition/checked/slots/default budgets hold. Defaults/checks/nullable views/Patch/nominal recursion/warnings and accepted API/wire/state remain. Composed/hierarchy/count/root-default and wider graph/platform roles stay separate.

[Unique composed-owned nested DTO intersections](composed-dto-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under allOf/union lexical owners, using original source carriers and existing deferred unique-owner proof. Variant roots/common fields/hierarchy/ambiguous/discarded/scratch/count/root-default roles stay separate. Canonical recursion, nullable/Patch/default/check/override/warning contracts and shared16384work/8MiB capture/source/checked/JVM240 limits remain; cache identity rejects foreign Pointers/catalogs. Existing API/wire/locks remain stable; wider graph/platform completion stays open.

[Named hierarchy leaf nested DTO intersections](hierarchy-dto-intersections.md) extend D01/D02/F04/F07 only to actual ordinary nested DTOs directly owned by named plain-source hierarchy leaves. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become nested payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings, shared16384work/8MiB capture and source/checked/JVM240 caps hold. Root/common/inline/composed-leaf/count/root-default/ambiguous roles stay separate; existing dispatch/API/wire/state remain stable, and broader graph/platform completion stays open.

[Named composed hierarchy-leaf nested DTO intersections](composed-leaf-intersections.md) extend D01/D02/F04/F07 to actual ordinary property/collection DTOs under named ordinary allOf leaf sources. Original catalog/Pointer and unique retained owner/capture proof remain; parent tags/common fields do not become payload fields or Consumer slots. Nullable/Patch/default/check/recursion/warnings and shared16384work/8MiB/source48/checked16/JVM240 limits hold. Root/common/inline/ambiguous/discarded/count/root-default roles stay separate; existing dispatch/API/wire/locks remain stable. Broader graph/platform completion remains open.

[Nested allOf DTO intersection targets](allof-target-intersections.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under direct ordinary object owners. Original composed source/alias snapshots and final retained owner proof remain; shared16384work/8MiB capture, source48/checked16/JVM240 and default/value caps hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Composed/union/hierarchy lexical owners, products/count/root-default targets and broader graph/platform completion remain separate.

[Composed intersection targets across lexical owners](composed-target-owners.md) extend D01/D02/F04/F07 to actual unique nested ordinary allOf constructors under allOf/union owners and named plain/allOf hierarchy leaves. Original source/alias snapshots, final unique owner/capture proof and parenttag isolation remain; shared16384work/8MiB/source48/checked16/JVM240 and default/value limits hold. Nullable/Patch/default/check/recursion/override/warning/API/wire/locks keep their contracts. Ambiguous/discarded/common/inline/product/count/root-default roles and broader graph/platform completion remain separate.

[Ordinary allOf directional HTTP DTOs](allof-directional-models.md) extend D07/A02/A03 to operation-owned request/public-response views from already validated winning original fields and merged required entries. Private analyzer source metadata stays outside Model identity; copies gain no original-source authority or new graph admission. Later overrides, hidden/discarded validation, required/Patch/default/check/recursion and original Pointers remain; shared DTO/API/wire/locks stay stable. Hierarchy/union/product/raw/count and changed-nominal default boundaries, source48/checked16/JVM240/clone/default/value budgets and broader platform/graph completion remain separate.

[Mapped scalar-field DTO uniqueness](dto-unique.md) extends D05/M03/M07/M12/V05 to actual ordinary named/original-inline DTO constructors with direct builtin scalar getters. Typed API/codec/nullable/default/required/Patch behavior remains; constructor initializer-aware default keys match runtime Float/UUID/Instant equality and retain original value Pointers. Actual source/member/constructor proof and independent shared16384work/8MiB source plus16384 default-key work budgets remain; enforcing getter/type fingerprints use a bounded cached reference while provenance/default keys stay private. General nested/enum/alias/composed/hierarchy/projected/count/native roles and broader platform/graph completion remain separate.


[DTO enum-field uniqueness](dto-enum-unique.md) extends the ordinary mapped-getter profile to direct String enums and checked native enum fields. String keys use actual serialName, including known/Unrecognized collisions; native membership/bounds and floating signed-zero checks remain. Constructor-aware default equality, original vocabulary/member/Pointer and unique actual nested enum proof, nullable/required/Patch and source/default/JVM/enum caps hold. Field-reference/nested/composed/hierarchy/projected/count/native-extension roles and broader graph/platform completion remain separate. Existing accepted API/wire/locks remain stable.


[DTO scalar const-body uniqueness](dto-const-unique.md) extends ordinary mapped DTO keys to actual direct builtin scalar body getters. Existing decoding can preserve non-literal values, so keys include actual body values alongside constructor getters; literal defaults never replace runtime getter equality. Original source/member/actual body name/type/wire/initializer proof, constructor-aware default keys, scalar codec approximation, nullable/required/Patch and all source/default/serialized-slot caps remain. Structured/null/native-extension/reference/const-only/composed/projected roles and broader graph/platform completion remain separate. Existing accepted API/wire/locks remain stable.

[DTO getter portability](platform-dto-getters.md) records the exact four-input Data/Plain platform profile: JVM/JS/macOS ARM64 Native runtime under four Json settings, common metadata and Android AAR compilation. Direct scalar/enum/const getters, constructor defaults, decoded body values, nullable and mapped Float/UUID/Instant equality are bound to input/consumer/source hashes. No fourth-input Android device, complete feature/HTTPengine, other-host or minimum-version claim is implied.

[DTO JsonNull getter uniqueness](dto-null-unique.md) extends ordinary DTO mapped keys to direct plain/null-only enum constructor fields and explicit/inferred null const bodies. Actual JsonNull and Kotlin null getter states remain distinct despite existing nullable wire collapse. Original vocabulary/default/source/member/actual getter/body proof, required/Patch/codecs and all source/default/serialized-slot caps hold; no fabricated type/enum or global raw/structural authority is introduced. Nested/reference/composed/projected/count/const-only and broader platform/graph roles remain separate. Existing accepted API/wire/locks remain stable.

[DTO inline scalar collection getter uniqueness](dto-collection-unique.md) extends ordinary mapped DTO keys to original finite inline List/typed Map builtin-scalar fields. Typed tree equality preserves List order/Map keys, Float zero and UUID/Instant mapping; constructor initializer expressions/null policy and canonical default keys/work bind to actual retained getters. Current outer edges plus getter edges share checked16, and nested initializer/value work shares16384 without resets. Original source/member/Pointer/capture/serialized-slot/default/API/wire/state contracts hold; refs/aliases/enum/JsonNull/DTO/raw/composed leaves and broader graph/platform completion remain separate.

HTTP compilation partitioning resolves actual declared symbols for all package roots, including aliases and quoted exports. Known sealed families retain their required common module; source inventory and registry assertions remain complete. The source-size packing target and compiler memory settings do not establish a generator resource or feature support limit.

[Null const object-default members](null-const-object-defaults.md) extend existing ordinary inline/named/allOf/finite object walks and component/wrapper factory values to original explicit/inferred null const properties. Actual JSON null is required for supplied values; omission/required/initializer suppression/Patch/no-inheritance and original Pointers retain their contracts. JsonNull body APIs/codecs remain non-null even with nullable annotations; original nodes and whole source/value/serialized-slot budgets remain. No global union pin/equality or unadmitted graph role is granted. Existing accepted API/wire/locks are stable.

[Known false-class JVM descriptors](false-nominal-jvm-limits.md) extend D08/E07 with a post-transform one-sided limit using actual retained false classifiers and direct nominal fields. Constructors and Data static copy descriptors use separate erased bounds; Plain and unknown aliases/value classes retain their own rules. Previously emitted sources with provably oversized constants now fail `E_MODEL_LIMIT` at the original Pointer with old output intact. The 240-slot cap and existing generic/resource probes remain; general feasibility/platform coverage stays open.

[Known false-class JVM generic signatures](false-generic-jvm-limits.md) extend D08/E07 with post-transform constructor and instance Data copy bounds from actual retained false classifiers. Inline List/Map arguments and proven non-null false List variance contribute conservative lengths; static copy uses its separate erased descriptor. Previously emitted uncompilable inputs now fail `E_MODEL_LIMIT` at the original Pointer with old output intact. Plain independence, unknown aliases/value classes, all original caps and resource probes remain; general feasibility/platform coverage stays open.

[Owner defaults containing literal false collections](literal-false-collection-root-defaults.md) extend D03/E03 for actual ordinary named Shared component-own annotations. Root factories borrow the original retained field/helper/false-leaf codec, validate required keys and independent field annotations, and preserve canonical complete literal-node identity including metadata-named JSON data. Empty/nested/nullable containers retain normal presence; false leaves reject every value. No default inheritance, cap, API/wire or persisted-format change is introduced for accepted inputs; other source/default/platform roles remain independent.

[Field defaults targeting literal false collection DTOs](literal-false-collection-dto-field-defaults.md) extend D03/E03 within the existing actual deferred consumer/target scope. Original leaf/Pointer/model/helper/TypeRef and independent target factories/counts remain authoritative; complete canonical consuming literals preserve metadata-named JSON data. Required suppression, nearest nullable, fresh typed initializer state and no target-default inheritance retain their contracts. All source/work/value/default/checked/slot/member caps and accepted API/wire/state formats remain; other source/default/platform roles stay independent.

[Owner-root defaults borrowing false DTO members](false-dto-owner-root-defaults.md) extend D03/E03 for original ordinary named Shared root annotations and direct unannotated reference members. Actual owner/getter/ref/target/helper/type/count identity and complete canonical literal capture remain bound; independent target factories validate after the full factory loop regardless of component naming order. Required keys, nearest nullable, strict false absence, normal collection presence and no default inheritance retain their contracts. All source/work/value/default/checked/slot/member caps and accepted API/wire/state formats remain; broader graph/source/platform roles stay independent.

[Pure DTO aliases in owner-root defaults](false-dto-owner-alias-defaults.md) extend D03/E03 through original finite named component paths. Canonical target/getter/serializer identity, nearest alias nullable then field override, original binding/height/cache/source snapshots and late independent factory/count/literal proof remain. Repeated members share immutable alias captures within the existing cumulative budget; source depth counts every hop. Existing caps and accepted API/wire/state formats remain, with alias-own/default-graph/context/platform roles independent.
