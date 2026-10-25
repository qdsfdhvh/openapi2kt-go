# DTO getter platform probe

The platform runner includes a fourth fixed synthetic model input,
[dto-getters.json](../tests/fixtures/platform/dto-getters.json), alongside the
original catalog, model-increments and roadmap-increments. Its
[common consumer](../samples/kotlin-platform-smoke/src/commonMain/kotlin/org/example/platform/DTOGetterContracts.kt)
checks named/inline ordinary DTO uniqueItems under validation and combined
optimized/locked options. The earlier three consumers retain all assertions.

The fourth input covers builtin scalar getters, constructor-initializer defaults,
open String enum known/Unrecognized serialName equality, native membership,
nullable fields/elements, Float signed zero/non-finite values, UUID/Instant
mapped equality and scalar const-body values changed by decoding. Const bodies
retain their existing scalar codec approximation; getters compare actual values.
All runtime profiles execute encodeDefaults × explicitNulls combinations.

| Target | Data | Plain | Evidence |
|---|---|---|---|
| JVM/JDK21 | pass | pass | Fresh compile and common runtime assertions |
| JS/Node | pass | pass | Fresh compile and common runtime assertions |
| macOS ARM64 Native | pass | pass | Fresh executable and common runtime assertions |
| Common metadata | pass | pass | Compilation only |
| Android AAR | pass | pass | AGP8.13.2, compileSdk36/minSdk23 assembly only |

Runs used Go1.27.2, Kotlin2.4.10, serialization1.11.0, Gradle9.5.1,
ZuluJDK21.0.12.1 and Node26.10.0 on macOS ARM64. Native used an installed
Xcode27.2 Beta3 selected only for the process. Dependency caches were prepared
before these offline runs; no build cache was used and all generation included
readonly checks. The final generator independently passed40 readonly checks
against the four generated roots in all10 compiled target/style projects.
[Source/input/consumer hashes and profile inventories](benchmarks/dto-getter-platforms.json)
bind the evidence to these exact inputs and generator source.

Run `bash scripts/kotlin/platforms.sh TARGET [data_class|plain_class]` with the
existing prerequisites in the [platform consumer guide](../samples/kotlin-platform-smoke/README.md).
JVM/JS/Native runtime, metadata compilation and Android assembly are distinct.
The fourth input has no Android device runtime evidence. Historical three-input
API34 device results do not establish it. Full independent model/HTTP/SSE/engine
matrices, other hosts, minimum dependencies and maximum resource combinations
remain separate acceptance work. This probe changes no generator API, wire or
persisted format.
