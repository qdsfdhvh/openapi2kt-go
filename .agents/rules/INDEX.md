# Task routing

Read the matching rows before making changes. Paths below are relative to this file; skills are loaded on demand, not all at session start.

| Task | Read |
|---|---|
| Using the CLI, asking about current supported inputs/features, integrating generated Kotlin, or troubleshooting generation/output errors | [Product usage skill](../../skills/openapi-kt/SKILL.md) |
| Go implementation, Go dependency changes, or review of Go code | [Go rules](go/go.md) |
| Parser/normalization, naming, generated Kotlin, serialization, CLI, or output changes | [Generator change skill](../skills/generator-change/SKILL.md) |
| Profiling, latency, memory, generated-code size, or performance claims | [Generator performance skill](../skills/generator-perf/SKILL.md) |
| Architecture or milestone design, including designing/implementing YAML support | [Development plan](../../docs/development-plan.md) |
| New fixture/sample, product documentation, attribution, packaging, or publication | [Contributing](../../CONTRIBUTING.md) |
| Agent instructions, skills, Claude settings, or symlinks | [Agent setup guide](../README.md) |

For Git writes and operations outside this repository, apply the authorization constraints in [AGENTS.md](../../AGENTS.md). Use only the PR/release automation actually configured in this repository.
