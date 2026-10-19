#!/usr/bin/env bash
# Explicit development dependency acquisition; platforms.sh stays offline.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
if [[ $# -lt 1 || $# -gt 2 || ( $# -eq 2 && "$2" != --offline ) ]]; then
  printf 'Usage: bash scripts/kotlin/prepare-platform-dependencies.sh {jvm|metadata|js|native|android} [--offline]\n' >&2
  exit 2
fi
platform_target="$1"
flags=()
if [[ $# -eq 2 ]]; then flags+=(--offline); fi
case "$platform_target" in
  jvm) task=runPlatformContracts ;;
  metadata) task=compileCommonMainKotlinMetadata ;;
  js) task=jsNodeDevelopmentRun ;;
  native) task=runDebugExecutableMacosArm64 ;;
  android) task=assembleDebug ;;
  *) printf 'Unsupported platform target\n' >&2; exit 2 ;;
esac
mkdir -p "$root/build"
workspace="$(mktemp -d "$root/build/platform-dependencies-$platform_target-XXXXXX")"
cp samples/kotlin-platform-smoke/{build.gradle.kts,settings.gradle.kts,gradle.properties} "$workspace/"
if [[ "$platform_target" == android ]]; then
  cp samples/kotlin-platform-smoke/android.gradle.kts "$workspace/build.gradle.kts"
  cp samples/kotlin-platform-smoke/android-settings.gradle.kts "$workspace/settings.gradle.kts"
  mkdir -p "$workspace/src/main"
  printf '<manifest />\n' > "$workspace/src/main/AndroidManifest.xml"
fi
mkdir -p "$workspace/src/commonMain/kotlin/org/example/platform"
python3 - "$workspace" "$platform_target" "${flags[*]}" <<'PY'
from pathlib import Path
import sys
project = Path(sys.argv[1])
if sys.argv[2] == 'js' and sys.argv[3] != '--offline':
    build = project/'build.gradle.kts'
    source = build.read_text()
    marker = '        args.add("--offline")\n'
    if source.count(marker) != 1:
        raise SystemExit('Expected one explicit offline npm flag in the platform build')
    # Online preparation warms the package cache, retaining --ignore-scripts
    # and installed Node/Yarn. The actual platform runner always stays offline.
    build.write_text(source.replace(marker, ''))
(project/'src/commonMain/kotlin/org/example/platform/PlatformContracts.kt').write_text('''package org.example.platform
import kotlinx.serialization.Serializable
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
@Serializable
private data class DependencyProbe(val value: String, val bytes: ByteArray)
fun main() {
    val value = Json.decodeFromString<DependencyProbe>("""{"value":"ready","bytes":[-128,127]}""")
    check(value.value == "ready" && value.bytes.contentEquals(byteArrayOf(-128,127)))
    check(Json.decodeFromString<DependencyProbe>(Json.encodeToString(value)).value == "ready")
    println("Platform compiler and serialization dependency preparation passed")
}
''')
PY
"${GRADLE:-gradle}" --no-daemon --no-build-cache -p "$workspace" \
  -PplatformTarget="$platform_target" "$task" "${flags[@]}"
printf 'Platform dependency preparation passed: %s %s\n' "$platform_target" "$workspace"
