# Generated Kotlin platform probe

This consumer checks a freshly generated synthetic catalog on a selected target. It is an additional portability probe, not the complete model/HTTP/SSE feature matrix and not a minimum-dependency guarantee.

Use an installed Go 1.27.2 toolchain, Gradle 9.3.1 and JDK 21. Select existing executables with `GO`, `GOFMT` and `GRADLE`; make the matching Go executable available on `PATH` for subprocess tests. A Go shim that relies on automatic toolchain switching does not suffice when `GOTOOLCHAIN=local` is required.

```bash
bash scripts/kotlin/platforms.sh jvm
bash scripts/kotlin/platforms.sh metadata
bash scripts/kotlin/platforms.sh js
bash scripts/kotlin/platforms.sh native
bash scripts/kotlin/platforms.sh android
# Repeat a target with explicit plain-class output:
bash scripts/kotlin/platforms.sh jvm plain_class
```

Each invocation builds the local CLI, generates into a fresh ignored workspace, verifies `--check`, and runs Gradle offline without the build cache. The input is `tests/fixtures/catalog.json`; dependencies are Kotlin/serialization plugin 2.4.10 and kotlinx.serialization 1.11.0. JVM bytecode targets Java 17. Prepare dependencies explicitly before invoking the offline check; missing caches are failures, not skipped tests. `GRADLE_USER_HOME` can select an isolated cache. The JS probe uses already installed Node and Yarn executables, disables their downloads, and passes `--offline --ignore-scripts` to dependency installation. `YARN_CACHE_FOLDER` can select a prepared cache; missing npm packages fail before Node execution.

| Target | Evidence when the command passes |
|---|---|
| `jvm` | Common-source JVM compilation and four `encodeDefaults × explicitNulls` runtime cases |
| `metadata` | Common metadata compilation with JVM and macOS ARM64 targets; no runtime evidence |
| `js` | Node execution of freshly compiled Kotlin/JS and the same four JSON cases |
| `native` | Actual macOS ARM64 executable and the same four JSON cases; requires that host and the matching Kotlin/Native toolchain |
| `android` | Android library compilation/assembly with AGP 8.13.2, compile SDK 36 and min SDK 23; requires the Android SDK. No device/runtime assertion is implied |

The fixed Kotlin 2.4.10 / serialization 1.11.0 / Gradle 9.3.1 combination has passed this catalog probe in both `data_class` and `plain_class`: JVM on JDK 21, JS/Node and macOS ARM64 Native execution, common metadata compilation, and Android AAR assembly with AGP 8.13.2 / SDK 36. These results establish this consumer profile only; Android device execution, other Native hosts, full platform feature matrices and minimum-version combinations remain unverified.

The runtime assertions exercise required scalar fields, nullable collection elements, recursive nullable model references, nested objects and raw JSON retention. They do not establish HTTP engines, SSE cancellation, all supported model roles, Android device behavior or other Native targets. Android runtime acceptance requires a separately executed real-device consumer. Full JVM feature acceptance remains `bash scripts/test.sh all`.
