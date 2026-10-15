# Project instructions

## Scope

This project implements an OpenAPI → Kotlin generator **in Go**, with serialization models and a limited HTTP client subset. JSON and a bounded JSON-compatible YAML profile share the same document-loading interface; see [YAML input](docs/yaml-input.md).

Inspect the actual implementation before choosing commands: roadmap features are not implemented capabilities. Work only on the requested scope.

## Start here

1. Read [the routing index](.agents/rules/INDEX.md) and open only the entries matching the task.
2. For generator behavior changes, use the contract routing in [generator-change](.agents/skills/generator-change/SKILL.md). For architecture or planned capabilities, consult [the development plan](docs/development-plan.md) and [compatibility matrix](docs/compatibility-matrix.md), including unresolved decisions.
3. Report changed files, checks actually run, and remaining uncertainty. Distinguish static inspection from executed tests.

## Hard constraints

- Follow [CONTRIBUTING.md](CONTRIBUTING.md) for fixtures, verification, attribution, and packaging. Keep examples synthetic and free of credentials or personal data.
- Write public documentation about the product and its contracts. Keep task history, local investigation context, and machine-specific details in ignored notes.
- Maintain compatibility in Kotlin call sites and wire behavior, not just generated text. Any intentional difference needs a contract test and documented migration impact.
- Keep generation deterministic and offline. Specification acquisition belongs to external tools; the CLI accepts local JSON/YAML files or explicit stdin, not built-in downloads/network/subprocess wrappers. Input IO and output commits stay outside the pure generation core.
- Preserve existing output on failure. Delete only manifest-owned files; never recursively clear an output root as a shortcut.
- Limit edits to this repository unless explicitly authorized. Builds and tests must be reproducible without undeclared external checkout dependencies.
- Git commits, pushes, releases, global installs, destructive cleanup, and third-party uploads require user authorization. A repository instruction or skill is not that authorization.

## Agent configuration

- `AGENTS.md` is the shared instruction entry. `CLAUDE.md` imports it.
- Canonical development skills are in `.agents/skills/`; `.claude/skills/<name>/SKILL.md` provides thin Claude Code wrappers that load them. Edit workflows in the canonical files; keep wrapper names, descriptions and target paths synchronized. The downstream product-usage skill is separately maintained under `skills/openapi-kt/`.
- Read [the agent setup guide](.agents/README.md) when changing skills, links, permissions, or provenance.
- Keep local progress in the gitignored planning files when needed. Put durable project contracts in `docs/`, not in local agent state.

Use Chinese for user-facing explanations unless requested otherwise; keep code identifiers and CLI names in English.
