# Agent setup

## Shared instructions

[`AGENTS.md`](../AGENTS.md) is the shared entry point. [`CLAUDE.md`](../CLAUDE.md) imports it for Claude Code. Both routes use the same [task index](rules/INDEX.md).

Keep the entry point concise. Put task-specific rules behind links and load only the entries that match the current task.

## Layout

```text
AGENTS.md
CLAUDE.md                         # imports AGENTS.md
.agents/
  README.md
  rules/
    INDEX.md                      # only file at the rules/ root
    go/
      go.md
  skills/
    generator-change/SKILL.md
    generator-perf/SKILL.md
  third-party/                    # attribution and required licenses
.claude/
  settings.json
  skills/
    generator-change/SKILL.md     # thin wrapper loading the canonical skill
    generator-perf/SKILL.md       # thin wrapper loading the canonical skill
```

New rules belong under `rules/<topic>/<topic>.md`, with a matching routing entry. Keep `rules/` itself limited to `INDEX.md` and topic directories.

## Skills

`.agents/skills/` is canonical for repository development skills. The downstream product-usage skill lives separately under [`skills/openapi-kt/`](../skills/openapi-kt/SKILL.md); see its [usage and packaging guide](../skills/README.md). Keep that directory self-contained when copied into a consumer project, and synchronize it with CLI contracts when behavior changes.

`.claude/skills/<name>/SKILL.md` files are thin Claude Code wrappers, not symlinks or copies of the workflows. Each wrapper declares the canonical skill's `name` and `description`, then instructs the agent to read and follow the canonical file. Resolve links inside that file from its canonical directory, not the wrapper directory. Keep wrapper metadata and target paths synchronized when adding or renaming skills. Hosts that discover `.agents/skills` can load skills directly; other hosts can follow the routing index explicitly.

Each skill needs `name` and `description` frontmatter, task triggers, repository-relative links, and verifiable completion criteria. Check attribution before adapting third-party material; retain required notices under `third-party/`.

Wrappers use ordinary files and repository-relative links, so checkout does not require symlink support. Automatic skill discovery and loading are host-dependent; verify them in the intended agent rather than assuming a filesystem check proves registration.

## Claude settings

`settings.json` uses the default permission mode. It grants no blanket shell access, configures no hooks or external services, and does not bypass approval prompts. Local overrides belong in the ignored `settings.local.json`.

These instructions describe expected agent behavior; they are not a security sandbox. Keep operational approvals explicit.

## Checking changes

Verify wrapper target links, matching skill names/descriptions, settings JSON, and that shared rules remain trackable while local state stays ignored. Write product-facing instructions rather than session journals. Distinguish executed Go/Kotlin checks from static inspection and cross-platform compilation.
