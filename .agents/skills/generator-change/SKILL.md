---
name: generator-change
description: Use when modifying OpenAPI parsing/analysis, generated Kotlin APIs or wire behavior, CLI behavior, or output ownership and transactions.
---

# Generator changes

## Select the contract

Read only the implemented contract relevant to the change:

| Surface | Contract |
|---|---|
| Core API, input limits, diagnostics, CLI or output ownership | [P1](../../../docs/p1-contract.md) |
| File/stdin IO or local configuration | [Local input](../../../docs/local-input.md), [configuration](../../../docs/configuration.md); acquisition is external, not a download adapter |
| Model types, naming, enums, unions or dictionaries | [Model semantics](../../../docs/model-semantics.md) |
| Date, byte or native scalar mappings | [Special scalars](../../../docs/special-scalars.md) |
| HTTP operations, parameters, responses or clients | [HTTP subset](../../../docs/http-client.md) |
| SSE event envelopes, payloads or Flow lifecycle | [SSE subset](../../../docs/sse.md) |
| Request-only defaults, missing/null or Patch | [Request presence](../../../docs/request-presence.md) |

For compatibility changes or new capabilities, consult the applicable [matrix entries](../../../docs/compatibility-matrix.md) and unresolved [design decisions](../../../docs/development-plan.md). Distinguish planned from implemented behavior; define the expected Kotlin call-site/wire result and any intentional difference.

## Scope the change and coverage

Reuse existing coverage; add a small synthetic fixture or inline case for uncovered behavior, following [fixture conventions](../../../CONTRIBUTING.md#test-fixtures). For a bug, reproduce the failure with an executable regression test before fixing it.

Prefer the shared resolver/analyzer/emitter decision point over per-output special cases. Check affected contexts, not every possible feature:

- component versus inline; model versus request, response or parameter;
- root versus field, list element or dictionary value;
- required/optional, nullable/non-null, default/const and nested/merged declarations.

Keep core generation pure and preserve output on failure. Public renaming is a compatibility change; persisted-format versions change only when their format/interpretation requires migration.

## Select evidence and finish

Follow the required [verification policy](../../../CONTRIBUTING.md#verification). The table adds behavior-specific evidence, not replacements for required checks.

| Changed behavior | Evidence |
|---|---|
| Kotlin type, literal, import or model shape | Go regression and freshly generated Kotlin compilation |
| Nullable/Patch/enum/union/custom serializer | Kotlin JSON encode/decode assertions for affected missing/null/unknown cases |
| HTTP parameter, request or response behavior | Fresh client compilation and controlled-engine URL/header/body/response assertions |
| Naming, deduplication or hoisting | Deterministic rerun, relevant options and Kotlin compilation |
| CLI flags, configuration or diagnostics | Actual parser/help/result/exit-code checks and synchronized docs/usage skill |
| Locks, ownership or output transactions | Filesystem/concurrency/failure-injection tests, deterministic checks and old-output preservation |
| File/stdin input boundary | Native file/pipe/EOF/8MiB/read-error/no-consume/check/name-lock parity and failure preservation; old download/config must reject without fetching, no curl wrapper |
| SSE lifecycle | Chunked-stream/cancellation tests; no broader support implied |

Review snapshots against the contract rather than bulk-accepting them. Kotlin changes alone do not require output failure injection; lock-only changes do not require Kotlin compilation unless they also change generated artifacts.

**Complete:** affected behavior is verified and contracts/docs are synchronized. Report changed files, checks and unavailable verification; intentional compatibility changes also need applicable matrix IDs and migration impact.
