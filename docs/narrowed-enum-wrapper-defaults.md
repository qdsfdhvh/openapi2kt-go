# Narrowed enum wrapper defaults

Retained String enum member names also have a separate [JVM name lower-bound check](enum-jvm-name-limits.md). Annotation size and generated member-name feasibility are independent.

An admitted component alias can own a non-null default even when its original enum vocabulary contains JSON null. The nearest explicit alias `nullable` setting controls the factory's return type and annotation:

```json
{
  "State": {"enum": ["on", null, "off"], "default": null},
  "Middle": {"$ref": "#/components/schemas/State", "nullable": true, "default": null},
  "Narrow": {"$ref": "#/components/schemas/Middle", "nullable": false, "default": "on"},
  "Again": {"$ref": "#/components/schemas/Narrow", "nullable": true, "default": null},
  "Bare": {"$ref": "#/components/schemas/Narrow"}
}
```

`State` and `Middle` retain their own null factories. `` `NarrowDto default`() `` returns the actual non-null `StateDto.ON`; `` `AgainDto default`() `` returns Kotlin null. `Bare` has no factory. Referencing fields keep their original required/optional/Patch behavior and do not inherit any component annotation.

This profile covers plain String enum sources with explicit nullable intent or [typeless String/null inference](implicit-string-enum-null.md), and already admitted native Boolean/Int/Long/Float/Double enums. Native aliases keep their validation opt-in, mapped scalar codecs and bounds. Finite pure named component chains use the [wrapper source and depth contract](reference-wrapper-component-defaults.md#finite-pure-component-alias-chains). New internal-reference paths, String formats/bounds, native overrides, value classes, const, compositions, null-only vocabularies and new field/child default roles retain their separate contracts.

## Source validation and values

The complete original source is checked with its original nullable grammar, vocabulary, member order, annotation and Pointer. Narrowing changes only the private consuming value plan. No null token is removed, no source schema is copied and no other component's default is substituted.

Each default must match the original vocabulary before numeric mapping or floating-point rounding and satisfy retained bounds. For example, a Float vocabulary containing `16777217` does not admit a wrapper annotation of `16777216`, even though both round to the same runtime Float. Factories retain original JSON numeric tokens and decode through the actual alias serializer.

A non-null alias rejects `default: null` and unknown enum values at its own `/components/schemas/Narrow/default`. Malformed source vocabularies and invalid source-owned annotations keep their original `/components/schemas/State/enum/...` or `/components/schemas/State/default` Pointers. Generation failure returns no artifacts and preserves owned CLI output and name locks.

Runtime policies remain per role: String enums keep `Unrecognized` for unknown strings, non-null aliases reject JSON null, and native scalar aliases use existing typed-consumer validation. A standalone builtin alias serializer is not a new membership validator.

Original source-depth 48, shared work/value 16,384, canonical/default 32 KiB, raw enum 240, checked depth 16, JVM serialized-slot 240 and input 8 MiB limits remain. These limits are independent: a long enum token can still produce an excessively long Kotlin identifier; a default that meets the byte budget does not establish arbitrary-name JVM feasibility.

## Compatibility

This D03/M03/M04/M07/M08 correction accepts eligible annotations that previously failed at a synthetic alias `/enum/<index>` when a copied validation view narrowed the source null grammar. Existing accepted call sites, generated files, serializers, warnings and persisted output/name-lock formats need no migration. A new accepted factory is opt-in; adding it does not alter constructor defaults or wire data. Broader defaults, reference graphs, platform acceptance and final release gates remain open.

## Verification

- [Core value/source/cap contracts](../core/narrowed_enum_wrapper_defaults_test.go) compare complete reports with the corresponding source vocabulary without null and retain strict/null/precision/bounds refusals.
- [Original-source identity contracts](../internal/schema/enum_wrapper_defaults_test.go) preserve source/member/annotation nodes and refuse copied, unrelated or incorrectly bound consumers and targets.
- [Native CLI contracts](../internal/cli/narrowed_enum_wrapper_defaults_process_test.go) cover read-only generation, optimizer name locks, deterministic replay, exact diagnostics and failure preservation.
- [Bounded fuzz coverage](../core/narrowed_enum_wrapper_defaults_fuzz_test.go) checks nullable inference parity, independent annotations, strict values and failure atomicity.
- `bash scripts/test.sh feature narrowed-enum-wrapper-defaults` freshly generates and executes Data/Plain × validation-off/on projects. It calls actual factories and preserves the existing String-null-member JSON/Patch/HTTP/SSE consumer, with four Json configurations, native narrowed request/response/SSE checks, nearest true/false, no inherited factory, nonempty locked optimizers, 239 Strings plus null, a 32 KiB default, 48 finite aliases and an actual 240-field constructor.
