# Retained enum member JVM names

The generator rejects a source-backed String enum member when its final generated member name alone exceeds the JVM constant's 65,535-byte UTF-8 capacity. Enum members become nested classes, so that member name is a lower bound of the actual class binary name. The check runs on retained declarations after deduplication, hoisting and merging, for both Data and Plain output and either validation setting.

For example, a String token containing 32,766 slashes expands through the existing operator naming rules to a 196,595-byte member name. Generation returns `E_MODEL_LIMIT` at the original `enum` array member. A nullable vocabulary's omitted null entry still counts when locating that Pointer. A same-size token containing spaces maps to the existing `VALUE` spelling and passes this check; source token size and emitted name size are different limits.

Temporary vocabularies used only for HTTP scalar projection and overwritten `allOf` fields do not emit member classes and do not trigger this check. Derived union/discriminator vocabularies without a direct original enum source remain outside this profile. Source-backed nested enums in retained models, requests, responses, errors and SSE payloads use their original source Pointers.

This is a conservative failure proof. Passing it does not guarantee general JVM feasibility: package and enclosing names, modified UTF-8 encoding, filesystem component limits, signatures, methods and other compiler resources have separate constraints. Existing source, value, enum-member, constructor-slot and checked-depth budgets still apply.

Previously, provably oversized members could produce Kotlin that failed during classfile emission. Those inputs now fail before output is committed; existing generated files and name locks remain intact on both generation and `--check` failure. Shorten the source token or revise its schema when this diagnostic occurs. The generator preserves public member spelling and existing accepted API, wire and state formats.

`bash scripts/test.sh feature narrowed-enum-wrapper-defaults` compiles and runs fresh Data/Plain × validation profiles, retaining the 32-KiB annotation, 240-member/field and 48-reference probes. It inspects the compiled `VALUE` class binary-name constant and checks the original slash-token refusal with the same producer. Core role/optimizer and native CLI tests cover deterministic diagnostics and failure preservation. Broader D08 and platform acceptance remain separate.
