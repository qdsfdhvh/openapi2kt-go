# openapi-kt usage skill

This directory contains an agent skill for **downstream Kotlin projects**. It teaches agents to validate OpenAPI JSON from local files or explicit stdin streams, generate Kotlin models and supported Ktor clients, check drift, and handle errors safely. Specification acquisition belongs to external tools, not a built-in downloader.

This package is for consuming the product, not modifying generator internals. Local generator-development agent configuration is not included in this package.

## Use

Start at [`openapi-kt/SKILL.md`](openapi-kt/SKILL.md). Copy the entire `openapi-kt/` directory, including `references/`, into your agent's supported project-level skill directory, such as `.agents/skills/openapi-kt/` on hosts supporting that convention. Copying only the main file breaks reference links. Automatic discovery is host-dependent; provide the explicit `SKILL.md` path when needed.

The skill includes no binary, installs no Go/Kotlin dependencies, and does not automatically generate files. Published CLI prereleases and source builds are described in the repository [installation guide](../docs/installation.md); use an existing user-selected executable or the documented source build. There is no verified remote skill publication URL, so no remote installation command is advertised here.

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

The usage reference also covers [inline ordinary DTO tuples](openapi-kt/references/kotlin.md#inline-dto-tuples): actual context codecs, isolated discarded-source validation, own annotations and approximation boundaries.

[Named Boolean tuple references](openapi-kt/references/kotlin.md#named-boolean-tuple-references) cover real false/nullable/raw codecs, independent annotations and original erased-prefix source budgets.

[Inline allOf tuples](openapi-kt/references/kotlin.md#inline-allof-tuples) document actual constructor/default codecs, discarded-source validation and shared overlap limits.

[Collection tuple sources](openapi-kt/references/kotlin.md#collection-tuple-sources) cover actual List/Map codecs, independent defaults, final backings and original source budgets.

[JsonNull tuple sources](openapi-kt/references/kotlin.md#jsonnull-tuple-sources) retain actual singleton/nullable codecs, strict outer defaults and original source budgets.

[Scalar const tuple sources](openapi-kt/references/kotlin.md#scalar-const-tuple-sources) retain actual codecs, strict original pins, approximation warnings and cumulative source binding budgets.

[Structured const tuple sources](openapi-kt/references/kotlin.md#structured-const-tuple-sources) retain actual JSON kind codecs, strict complete-literal defaults and original data-only source capture.

The downstream usage bundle includes a self-contained [union tuple reference](openapi-kt/references/tuple-union-sources.md) for original-source, context-codec and own-default boundaries.

The downstream usage bundle also includes [product union tuples](openapi-kt/references/tuple-product-union-sources.md), covering actual composition dispatch and original-source/default boundaries.

The downstream bundle includes [hierarchy tuple references](openapi-kt/references/tuple-hierarchy-references.md), preserving actual named lineage/codec and annotation boundaries.

The usage bundle includes [recursive cover tuple references](openapi-kt/references/tuple-recursive-cover.md) for actual JSON/null codecs, outer annotations and original graph budgets.

The usage bundle includes [special tuple references](openapi-kt/references/tuple-special-references.md) for actual scalar/provider/JSON codecs, strict outer annotations and shared named-alias capture budgets.
