# Generated Kotlin platform probe

This consumer checks a freshly generated synthetic catalog on a selected target. It is an additional portability probe, not the complete model/HTTP/SSE feature matrix and not a minimum-dependency guarantee.

Use an installed Go 1.27.2 toolchain, Gradle 9.5.1 and JDK 21. Select existing executables with `GO`, `GOFMT` and `GRADLE`; make the matching Go executable available on `PATH` for subprocess tests. A Go shim that relies on automatic toolchain switching does not suffice when `GOTOOLCHAIN=local` is required.

```bash
bash scripts/kotlin/platforms.sh jvm
bash scripts/kotlin/platforms.sh metadata
bash scripts/kotlin/platforms.sh js
bash scripts/kotlin/platforms.sh native
bash scripts/kotlin/platforms.sh android
# Repeat a target with explicit plain-class output:
bash scripts/kotlin/platforms.sh jvm plain_class
```

Each invocation builds the local CLI, generates into a fresh ignored workspace, verifies `--check`, and runs Gradle offline without the build cache. It retains the original `tests/fixtures/catalog.json` with default options and separately generates [model-increments.json](../../tests/fixtures/platform/model-increments.json) with validation and model merging enabled. The second input checks allOf winning defaults, signed-byte default trees/factories and owned self-recursive aliases. Both outputs compile as common sources. Dependencies are Kotlin/serialization plugin 2.4.10 and kotlinx.serialization 1.11.0. JVM bytecode targets Java 17. Prepare dependencies explicitly before invoking the offline check; missing caches are failures, not skipped tests. `GRADLE_USER_HOME` can select an isolated cache. The JS probe uses already installed Node and Yarn executables, disables their downloads, and passes `--offline --ignore-scripts` to dependency installation. `YARN_CACHE_FOLDER` can select a prepared cache; missing npm packages fail before Node execution.

Explicit preparation is separate from verification:

```bash
bash scripts/kotlin/prepare-platform-dependencies.sh js
bash scripts/kotlin/prepare-platform-dependencies.sh native
# Existing-cache preparation can itself be checked offline.
bash scripts/kotlin/prepare-platform-dependencies.sh jvm --offline
```

The preparation command accepts the same target names, compiles a small synthetic serialization probe and can acquire declared Gradle/npm dependencies. It consumes no OpenAPI document. Online JS preparation retains `--ignore-scripts` and uses installed Node/Yarn; the actual platform runner always retains offline package installation. Android SDK installation is separate and automatic SDK download is disabled. An incomplete cache or inaccessible repository is a failed prerequisite, with no runtime acceptance implied. Verification never invokes preparation automatically.

| Target | Evidence when the command passes |
|---|---|
| `jvm` | Common-source JVM compilation and four `encodeDefaults × explicitNulls` runtime cases |
| `metadata` | Common metadata compilation with JVM and macOS ARM64 targets; no runtime evidence |
| `js` | Node execution of freshly compiled Kotlin/JS and the same four JSON cases |
| `native` | Actual macOS ARM64 executable and the same four JSON cases; requires that host and the matching Kotlin/Native toolchain |
| `android` | Android library compilation/assembly with AGP 8.13.2, compile SDK 36 and min SDK 23; requires the Android SDK. No device/runtime assertion is implied |

The earlier catalog-only probe passed the fixed Kotlin 2.4.10 / serialization 1.11.0 / Gradle 9.3.1 combination in both `data_class` and `plain_class`: JVM on JDK 21, JS/Node and macOS ARM64 Native execution, common metadata compilation, and Android AAR assembly with AGP 8.13.2 / SDK 36. These results establish this consumer profile only; Android device execution, other Native hosts, full platform feature matrices and minimum-version combinations remain unverified.

The expanded two-input probe requires its own execution evidence and does not inherit those historical results. The runtime assertions exercise required scalar fields, nullable collection elements, recursive nullable model references, nested objects and raw JSON retention, plus winning allOf defaults/retained validation, byte-tree wire/fresh instances/factories, and canonical self-alias assignment/serde/descriptors. They do not establish HTTP engines, SSE cancellation, directional operation DTOs, all supported model roles, Android device behavior or other Native targets. Android runtime acceptance requires a separately executed real-device consumer. Full JVM feature acceptance remains `bash scripts/test.sh all`.

The expanded catalog + model-increments probe has also passed with Kotlin 2.4.10 / serialization 1.11.0 / Gradle 9.5.1 on macOS ARM64, in Data and Plain styles: JVM/JDK21 and JS/Node runtime, common metadata compilation, macOS ARM64 Native runtime, and Android AAR assembly (AGP8.13.2/SDK36). Native execution used an installed Xcode27.2 toolchain selected by a process-local `DEVELOPER_DIR`; no global Xcode selection is required by the generator. Dependency preparation was explicit and each acceptance run was offline. These results cover the two probe inputs and their four Json configurations; they do not establish all feature roles, Android device runtime or minimum versions.


The runner also preserves a third synthetic input,
[roadmap-increments.json](../../tests/fixtures/platform/roadmap-increments.json),
using validation and combined dedup/hoist/merge/lock options in its own package.
Its [common consumer](src/commonMain/kotlin/org/example/platform/RoadmapIncrementContracts.kt)
checks actual allOf constructor tuples and nominal recursion, independent
component/alias factories without default inheritance, open String-enum tuples,
scalar tuple field defaults/counts, structural List/Map uniqueness and preserved
primitive-union root/wrapper/Null aliases. The first two inputs and all their
assertions remain. The additional source directory is compiled by both the KMP
and Android builds.

This three-input profile passed Kotlin2.4.10/serialization1.11.0/Gradle9.5.1 on
macOS ARM64 in Data and Plain styles: actual JVM/JDK21, JS/Node and macOS ARM64
Native runtime with four Json configurations, common metadata compilation and
Android AAR assembly with AGP8.13.2/SDK36. Runs were offline against explicitly
prepared caches; Native used a process-local installed Xcode27.2 selection.
Android assembly has no device runtime assertions. These exact inputs and
options do not establish every model/HTTP/SSE role, other hosts/engines or minimum
dependency versions. Those remain independent acceptance work.

With the [JVM-name correction](../../docs/jvm-names.md), the same three-input Data/Plain profile also passed min SDK23 test APK DEX compilation and headless instrumentation on a physical Android API34 device. Both test packages were removed after execution. No Activity was launched and the test manifests contained no permissions, services, providers or receivers. This is actual API34 runtime evidence, separate from min SDK23 DEX compilation and Android AAR assembly; actual API23 runtime and full model/HTTP/engine matrices remain unverified.

### Headless Android device execution

Select an authorized connected device explicitly with `ANDROID_SERIAL`, and set
`ANDROID_HOME` (or `ANDROID_SDK_ROOT`) to an existing SDK. The `android-device`
target freshly generates/checks the current four inputs, builds a min SDK23 test
APK offline, audits its final manifest, runs the common assertions without
launching an Activity, and uninstalls only the package installed by that run.
It refuses to replace a preexisting probe package. The selected device must
already be authorized for adb; this command does not change device settings.

```bash
ANDROID_SERIAL=YOUR_DEVICE bash scripts/kotlin/platforms.sh android-device
ANDROID_SERIAL=YOUR_DEVICE bash scripts/kotlin/platforms.sh android-device plain_class
```

The SDK must already contain `platform-tools/adb` and
`cmdline-tools/latest/bin/apkanalyzer`; `ADB` and `ANDROID_APKANALYZER` can select
other installed executables. If the offline AGP aapt2 artifact is unavailable,
`ANDROID_AAPT2` can select an already installed SDK `aapt2` binary. Nothing is
downloaded automatically. An unavailable device, missing cache, failed contract
or failed cleanup fails the command. APK hashes, actual device API and raw
instrumentation output are saved beside the APK in the ignored workspace;
device serials are not included in the result JSON. The test APK's package must
not have permissions or Activity/service/provider/receiver components. DEX
compilation for min SDK23 and runtime on the selected API are separate results;
this runner does not imply coverage of every API or feature/HTTP engine.


The runner now also generates [dto-getters.json](../../tests/fixtures/platform/dto-getters.json)
in a fourth source root. Its common consumer checks direct DTO scalar/enum/const
getters, constructor defaults, actual decoded body values, nullable, floating
zero/non-finite values and UUID/Instant equality. The exact four-input Data/Plain
profile passed JVM/JS/macOS ARM64 Native runtime and metadata/Android AAR
compilation with the [recorded fixed versions and inventories](../../docs/platform-dto-getters.md).
The fourth input has no device runtime result; the earlier three-input API34
result remains historical. Full feature/HTTP/engine/minimum-version acceptance
is separate.


The runner also generates [enum-field-bounds.json](../../tests/fixtures/platform/enum-field-bounds.json)
in a fifth source root. Its common consumer checks direct lexical String enum
field bounds and metadata on existing direct List/typed Map enum leaf bounds.
Known members and Unrecognized keep open standalone enum codecs; actual DTO
constructor/automatic decoding checks, strict compile-time defaults, nullable
views, required fields and fresh containers retain the product contract under
four Json configurations. Metadata uses String words without date/UUID/provider
dependencies. Original DTO root/alias/null factories and consuming object/DTO-list annotations
also retain independent values, no inheritance and fresh state. Both KMP and
Android projects include this source root.

The fifth input requires its own target/style runtime and compilation evidence;
it does not inherit the historical four-input or Android device results. Run
the same platform commands above. JVM, JS/Node and macOS ARM64 Native execute
the common assertions; metadata and Android AAR targets compile them. This
bounded probe does not establish the complete feature/HTTP/engine matrix,
minimum dependencies or actual Android API23 behavior.


The five-input Data/Plain probe has [separate fixed-combination evidence](../../docs/platform-enum-field-bounds.md):
JVM, JS/Node and macOS ARM64 Native common assertions execute under four Json
configurations, and common metadata plus Android AAR compile. Kotlin/plugin2.4.10
and serialization1.11.0 use Gradle9.3.1 for KMP and Gradle9.5.1 for Android.
No fifth-input Android device or complete feature/HTTP/engine acceptance is
implied.
