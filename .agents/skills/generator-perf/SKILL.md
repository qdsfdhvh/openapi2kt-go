---
name: generator-perf
description: Use when measuring generator or generated-Kotlin costs, optimizing a measured bottleneck, or verifying performance-regression and speedup claims.
---

# Generator performance

Preserve the adapted workflow's [attribution and MIT license](../../third-party/README.md).

## Choose the intent

| Request | Scope and completion |
|---|---|
| Measure a baseline | Reproducible workload, raw samples, summary statistics and limitations; no mandatory profiling or code change |
| Optimize a bottleneck | Baseline, evidence identifying the bottleneck, scoped change and correctness-checked before/after measurements |
| Verify a regression or speedup claim | Comparable workloads/options/environments and measurements supporting the exact claim, or an explicit inconclusive result |

For inventory/benchmark harness commands, see [validation and performance](../../../docs/validation-and-performance.md).

## Workload and measurements

Inspect the actual binary/options/build flags/harness. Use supplied accepted specs or synthetic workloads; record input hashes/dimensions and generated datasets' seeds. Keep private specs local. Every measured generation sample must pass expected artifact/manifest/hash or semantic checks. Separate early rejection from successful generation.

Measure only the applicable costs, separately when attribution matters:

- compiler cold/warm builds, noting whether OS caches remain warm;
- process startup, pure generation and end-to-end output commit;
- parsing/analysis/emission versus filesystem operations;
- peak RSS/allocations and binary/generated-source sizes;
- downstream Kotlin compilation time and artifact size.

Retain raw samples, toolchain/host/flags, warmups and sample counts. Distinguish fresh output from no-change regeneration. Go execution speed does not establish Kotlin compilation or artifact savings.

## Optimization only

Identify the measured bottleneck using available Go benchmarks, CPU/heap profiles or `go tool pprof` when needed. Tool installation requires approval. Keep build/profile/stripping flags explicit; let evidence guide algorithmic improvements, graph reuse or concurrency.

Preserve deterministic order, resource guards, output ownership and caller-specific Kotlin semantics. Code changes follow [contribution verification](../../../CONTRIBUTING.md#verification); emission/model behavior changes also follow [generator-change](../generator-change/SKILL.md).

## Comparisons and reporting

Compare correctness-checked inputs/options under equivalent host/cache conditions. Include repeated and mostly-unique shapes for reuse/caching claims. Compare timing distributions, not the best sample; separate configuration effects from algorithmic gains.

Keep raw logs/profiles in ignored directories and reproducible aggregates in product docs. Report inconclusive results and unverified platforms/comparisons without implying speedup or wire parity. Baseline-only requests end with the measurement report.
