# Avoiding discarded const initializer construction

Ignored structured const declarations validate their original schema syntax and JSON kind without constructing a temporary Kotlin initializer. Real body/value constants retain the existing JSON encoding and initializer path. The [discarded declaration contract](discarded-structured-const.md), source diagnostic order, warning policy, generated files and name-lock identities remain unchanged.

Previously, every model that borrowed an ignored structured declaration serialized the literal into JSON, quoted that JSON for Kotlin, and then discarded the expression. The generator now shares the original const syntax and structured-kind checks between validation and expression construction. The document loader still validates literal JSON and its independent input/depth budgets. No source cache, expanded schema copy, additional accepted grammar or changed cap is involved.

The following measurements use Go 1.27.2 on macOS ARM64, ordinary `data_class` generation, `Dto` suffix, no optimizer or validation flag, default heap settings and synthetic accepted inputs. Each case had one warmup and three measured runs. Native runs alternated before/after binaries sequentially and wrote fresh output. Pure-core runs measured parse/analyze/transform/emit and per-call allocated bytes; GC ran before each sample, outside timing. Every sample checked exact Kotlin and report equality against its projection control. The profiler identified temporary structured initializer construction as the allocation bottleneck.

| Workload | Native CLI median before → after | Pure-core allocated MiB before → after |
|---|---:|---:|
| 2 MiB ignored literal, one model | 25.3 → 13.6 ms | 30.2 → 10.0 |
| Same literal shared by 128 derived models | 1462.4 → 31.8 ms | 2609.8 → 13.5 |
| 64 distinct ignored literals, 32 KiB each | 33.8 → 23.6 ms | 22.9 → 3.3 |
| 64 distinct overridden const fields, 32 KiB each | 35.2 → 22.5 ms | 23.4 → 3.7 |

The shared-source native range was 1.43–1.48 s before and 31–37 ms after in this run. Projection-only controls remained in the same small-process cost range. These figures cover successful generator work, startup and fresh output commits; they do not measure cold builds, Kotlin compilation, downstream artifact savings, platform runtime support or peak RSS. Small timings are host/cache sensitive and three measured samples do not establish a general latency guarantee. Existing source48/work16,384/member240/default32KiB/checked16/input8MiB limits and independent boundary probes remain.

No API, wire or persisted-format migration is required. Run the existing `feature discarded-structured-const` contract for Data/Plain × validation, four JSON settings, required winners, real body/dictionary values, optimized locks, 240-member constructors, HTTP and fragmented SSE. Diagnostic precedence and original node preservation are checked in the [schema regression](../internal/schema/discarded_const_validation_test.go); shared and mostly-unique projection equality is checked in the [core regression](../core/discarded_const_validation_test.go). `go test ./internal/schema -run '^$' -bench 'Benchmark(DiscardedStructuredConstValidation|StructuredConstTemporaryInitializer)$' -benchmem` compares diagnostic-only validation with the retained temporary-initializer path on a 2 MiB literal. Use the repository's offline toolchain settings from [Contributing](../CONTRIBUTING.md#verification).
