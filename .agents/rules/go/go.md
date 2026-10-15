# Go development rules

## Core and adapters

- Keep parsing, normalization, analysis, naming and Kotlin emission behind `core.Generate` / `core.GenerateWithDiagnostics`. Pass document bytes/options in; return relative artifacts and diagnostics out.
- Filesystem, environment and subprocess access belong in CLI/output adapters or test tooling. Core tests exercise the same generation API as the CLI.
- Model absent, null, unsupported, scalar, object and union states explicitly. Preserve source pointers instead of copying expanded reference trees.
- Preserve declaration/dispatch order; sort unordered file/symbol inventories. Use deterministic, versioned hashes for persisted identities.
- Bound graph depth, expansion work, branch counts and JVM constructor slots. When graph traversal or caching changes, test recursive references and cached graph heights.

## Errors and dependencies

- Return safe structured errors for malformed input and user-controlled paths. Diagnostics contain codes/locations, not payloads or raw OS errors.
- Keep internal representations private. Share context-aware resolution and emission rather than adding per-output special cases.
- Fix vet/race findings at their source. Document narrow exceptions and OS-specific locking/unsafe adapters beside their implementation.
- Inspect `go.mod` and the installed toolchain before adding dependencies. Review licenses, maintenance, offline builds and platform impact; keep `go.sum` when modules require it.
- Kotlin output patterns and enum spellings follow the product contract, not Go regexp/Unicode defaults.

## Verification

For Go implementation changes, use affected tests during iteration and finish with the required [CONTRIBUTING verification](../../../CONTRIBUTING.md#verification), including formatting, vet, test/race and offline toolchain settings. Targeted tests do not replace required gates.

Use [generator-change](../../skills/generator-change/SKILL.md) for additional evidence: Kotlin changes need compilation/applicable runtime assertions; lock/ownership/transaction changes need filesystem/failure-injection coverage. Select by affected behavior, not unrelated surfaces.

Use bounded fuzz budgets when relevant to input or graph changes; retain failures and reproducers. Build-only cross-compilation is not runtime evidence. Documentation-only changes use relevant link/frontmatter checks, not an unrelated generation or Kotlin test suite.
