# openapi-kt usage skill

This directory contains an agent skill for **downstream Kotlin projects**. It teaches agents to validate OpenAPI JSON from local files or explicit stdin streams, generate Kotlin models and supported Ktor clients, check drift, and handle errors safely. Specification acquisition belongs to external tools, not a built-in downloader.

This package is for consuming the product, not modifying generator internals. Local generator-development agent configuration is not included in this package.

## Use

Start at [`openapi-kt/SKILL.md`](openapi-kt/SKILL.md). Copy the entire `openapi-kt/` directory, including `references/`, into your agent's supported project-level skill directory, such as `.agents/skills/openapi-kt/` on hosts supporting that convention. Copying only the main file breaks reference links. Automatic discovery is host-dependent; provide the explicit `SKILL.md` path when needed.

The skill includes no binary, installs no Go/Kotlin dependencies, and does not automatically generate files. The generator currently has no published binary installation package; use an existing user-selected executable or the documented source build. There is no verified remote skill publication URL, so no remote installation command is advertised here.

Example request after installing the skill:

> Use openapi-kt to validate ./openapi.json, generate into build/generated/openapi with package org.example.api and suffix Dto, and confirm consistency with --check.

For a read-only request:

> Use openapi-kt to check whether the generated output for ./openapi.json has drifted, without modifying files.

## Layout and maintenance

```text
skills/
  README.md
  openapi-kt/
    SKILL.md
    references/
      commands.md
      kotlin.md
```

Keep common decisions and safety boundaries in the main file. On-demand references use only links within the skill package, so the entire directory remains usable after copying. Update the skill and product contracts together when CLI flags, diagnostics, output layout or support boundaries change. Commands must match actual help and implemented contracts, not roadmap interfaces.
