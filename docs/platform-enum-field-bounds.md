# Lexical enum bounds platform probe

The [platform runner](../scripts/kotlin/platforms.sh) builds the local CLI,
generates five independent inputs, verifies the same-options `--check`, and
compiles fresh common consumers offline. Data and Plain models run the same
assertions under all four encodeDefaults/explicitNulls combinations.

| Input | Covered behavior |
|---|---|
| [catalog](../tests/fixtures/catalog.json) | Existing required/nullable/recursive models and raw JSON retention |
| [model increments](../tests/fixtures/platform/model-increments.json) | Winning allOf defaults, signed bytes and owned self aliases |
| [roadmap increments](../tests/fixtures/platform/roadmap-increments.json) | Tuple/default/uniqueness/primitive-union alias profile |
| [DTO getters](../tests/fixtures/platform/dto-getters.json) | Actual scalar/enum/const getter equality and defaults |
| [enum field bounds](../tests/fixtures/platform/enum-field-bounds.json) | Scalar and finite List/typed Map enum checks, metadata, structural uniqueness, independent object factories and checked internal enum borrowing |

The fifth input preserves String words under date/UUID/native metadata without
mapping those enum words to provider, date or UUID types. Known and Unrecognized values retain open
standalone enum lookup/codecs; DTO construction and automatic decoding check
actual serialName lengths and patterns. The common consumer checks valid unknown
words, invalid known/unknown payloads, required/default/missing/null behavior,
nullable metadata enums, fresh container defaults and JSON round trips. It also
checks the original DTO root factory, pure alias/null factories, consuming object
and DTO-list defaults, no inheritance and fresh state against the same lexical
field checks. Its nested List/Map field additionally covers independent nullable layers,
strict original default membership, valid unknown runtime words, map-order-independent
uniqueness, actual constructor/codec checks and fresh trees under the
[finite inline bounds contract](nested-string-enum-bounds.md).
Compile-time defaults remain strict under the [field bounds contract](string-enum-field-bounds.md).

| Target | Data and Plain evidence | Fixed combination |
|---|---|---|
| JVM | Common assertions executed, four Json configurations per style | Kotlin/plugin2.4.10, serialization1.11.0, Gradle9.3.1, JDK21; JVM bytecode17 |
| JS/Node | Common assertions executed, four Json configurations per style | Same Kotlin/serialization/Gradle; installed Node26.12.1 and Yarn1.22.22 |
| macOS ARM64 Native | Common assertions executed, four Json configurations per style | Same Kotlin/serialization/Gradle; installed Xcode27.2 beta3 selected per process |
| Common metadata | Compiled for the JVM/macOS ARM64 source set | Same Kotlin/serialization/Gradle; compilation only |
| Android AAR | Library assembled for both styles | Kotlin/plugin2.4.10, serialization1.11.0, Gradle9.5.1, AGP8.13.2, compileSDK36/minSDK23, installed AAPT2 36.0.0 |

Use the existing `GO` and `GRADLE` selectors for those installed versions. Run
`bash scripts/kotlin/platforms.sh TARGET STYLE` with targets `jvm`, `js`, `native`,
`metadata` or `android`, and styles `data_class` or `plain_class`.
`DEVELOPER_DIR` may select an installed complete Xcode for Native without changing
the global selection; `ANDROID_AAPT2` may select an installed SDK tool. Dependencies
must be prepared explicitly before acceptance. The runner keeps Gradle offline,
disables the build cache, reruns tasks, disables Node/Yarn downloads and uses
offline npm installation with scripts ignored. Missing artifacts fail; the
runner never automatically prepares dependencies.

These results cover the five exact inputs and their selected options. They do
not establish every feature, option interaction, HTTP engine, dependency minimum
or Native host. Metadata and AAR compilation provide no device/runtime evidence.
The fifth input has no Android device result; minSDK23 compilation does not
establish actual API23 behavior. Historical device checks for earlier inputs
remain separate. Full JVM feature acceptance and final candidate/release gates
retain their independent requirements.

The fifth input also includes [bounded collection roots/aliases](bounded-enum-collection-roots.md), their independent factories and consuming field/object/DTO-list defaults. Actual Record fields enforce their shared lexical checks while standalone root-alias codecs retain the open policy. Target and runtime evidence remain scoped to the exact input and toolchain.

[Checked lexical enum borrowing](borrowed-checked-string-enums.md) is exercised
through the original Record state and an unannotated Map/Map source. Pure aliases
retain actual nested classifiers; alias factories and consuming field annotations
are independent. Common assertions cover fresh defaults, missing/null/value,
known/unknown serialName checks, duplicate known/unknown words and typed JSON
round trips. Independent alias codecs remain open, while original and borrowed
DTO getters enforce checks. This profile adds no general enum-root, child-default
or HTTP-engine/platform admission.

The fifth input also exercises
[enum-reference collection factories](referenced-enum-collection-factories.md):
checked lexical String and Int, Long, Float, Double and Boolean enum sources,
independent source/alias/collection defaults, nullable factories, fresh containers,
consuming field defaults and unannotated fields. The String alias codec remains
open to unknown values. This expanded input passed in Data and Plain styles on
JVM, JS/Node and macOS ARM64 Native under all four Json configurations; common
metadata and Android AAR compilation also passed for both styles. These runs used
Go 1.27.2, Kotlin/plugin 2.4.10, serialization 1.11.0, Gradle 9.5.1 and JDK 21;
JS used Node 26.12.1 and Yarn 1.22.22, Native used a process-local Xcode 27.2
beta 3 selection, and Android used AGP 8.13.2 with SDK/AAPT2 36.0.0. The
generated factories and common consumer were also present in both AARs.
Android device runtime, API 23 runtime and the complete feature, engine and
dependency matrices remain separate.

The fifth input also includes independent inline and referenced enum child
annotations, including a nested referenced element, and direct builtin scalar,
UUID, LocalDate and Instant List/Map child annotations on ordinary DTOs with and without independent owner
factories. The common consumer checks factory values, fresh containers, missing
fields and supplied-value round trips. The
expanded Data/Plain input passed JVM, JS/Node and macOS ARM64 Native execution
under four Json configurations, and common metadata and Android AAR compilation
for both styles, using the fixed toolchain above and kotlinx-datetime 0.7.1. Android device runtime and
general child-default or platform acceptance remain separate.
