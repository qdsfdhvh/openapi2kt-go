# Implicit nullable typeless String enums

A plain typeless enum containing at least one actual String and one JSON null supplies intrinsic nullable intent when `nullable` is absent:

```json
{"enum":["ready",null,"done","null"]}
```

This has the same generated API, diagnostics and wire behavior as the existing `nullable: true` form. The physical enum classifier and its serializer remain non-null, with READY/DONE/NULL members and open `Unrecognized` values. Typed uses and reified nullable aliases handle JSON null. The String `"null"` remains a distinct enum member.

Explicit `type: "string"` or `nullable: false` retains its non-null refusal. Existing type-plus-null and explicit nullable behavior remain. Null-only vocabularies keep their separate JsonNull value contract. Inference does not cross format/native/value-class/const/composition/structural declarations or add a String member. Duplicate null/String entries and other malformed vocabularies still fail; raw 240-member limits count null.

The shared type query reads original vocabulary nodes without adding `type`/`nullable` members or copying expanded schemas. Actual root/internal/borrowed enum owners, original Pointers, alias overrides and retained declaration authority stay in force. Existing field/default/list/map/tuple, collection uniqueness, object-default, HTTP and SSE roles reuse the same source decision. This grants no additional default role or inheritance: non-null annotations still belong to the original String vocabulary, while runtime unknown Strings remain open. Required/optional/Patch and per-layer container nullability keep their existing policies.

Only actual String tokens contribute union tags; null adds no tag named `"null"`. Mapped hierarchy body tags retain their existing non-null type and dispatch. Inline operation String projections keep their scalar approximation, including the existing raw String response reader; no additional response codec is introduced.

Migration: these typeless mixed String/null inputs previously failed generation. Their new output matches the explicit nullable control. Previously accepted inputs retain their API, wire, name-lock identities and persisted formats. Existing source48/work16,384/raw-enum240/JVM240/checked16/default32KiB/input8MiB limits remain. General enum/composition inference, strict runtime membership and platform acceptance remain independent.

Run `bash scripts/test.sh feature implicit-string-enum-null` for fresh Data/Plain with validation off/on. It derives inferred inputs from existing synthetic fixtures and reuses their actual consumers: open/unknown enums, defaults, borrowed references, required/Patch, optimized locked replay, four Json settings, controlled HTTP and fragmented live SSE. It also compiles actual 240-field constructors, a raw 240-member enum, 15 containers and a 45-hop alias, and runs the existing mapped hierarchy tag codec. Checked Data/Plain projects additionally exercise all eight other existing nullable-vocabulary fixture consumers, including tuple references and mapped DTO/collection uniqueness.

The [core regressions](../core/implicit_string_enum_null_test.go) compare all 11 existing nullable-vocabulary fixtures with explicit controls and retained failure profiles, plus source height/member boundaries. [Original source authority](../internal/schema/string_enum_null_members_test.go) runs both explicit and inferred variants, including missing/replaced/repeated declarations, same-byte literal substitutions and alias/cache changes. [Native checks](../internal/cli/implicit_string_enum_null_process_test.go) preserve output and locks on malformed vocabulary/defaults; [bounded fuzzing](../core/implicit_string_enum_null_fuzz_test.go) compares inferred and explicit success reports and non-null failure atomicity. These checks establish this source profile, not the broader roadmap or full platform validation.
