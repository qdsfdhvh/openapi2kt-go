#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
if [[ $# -lt 1 || $# -gt 2 ]]; then
  printf 'Usage: bash scripts/kotlin/platforms.sh {jvm|metadata|js|native|android|android-device} [data_class|plain_class]\n' >&2
  exit 2
fi
platform_target="$1"
model_style="${2:-data_class}"
case "$platform_target" in
  jvm) task=runPlatformContracts ;;
  metadata) task=compileCommonMainKotlinMetadata ;;
  js) task=jsNodeDevelopmentRun ;;
  native) task=runDebugExecutableMacosArm64 ;;
  android) task=assembleDebug ;;
  android-device) task=assembleDebugAndroidTest ;;
  *) printf 'Unsupported target\n' >&2; exit 2 ;;
esac
case "$model_style" in data_class|plain_class) ;; *) printf 'Unsupported model style\n' >&2; exit 2 ;; esac
if [[ "$platform_target" == android-device && -z "${ANDROID_SERIAL:-}" ]]; then
  printf 'ANDROID_SERIAL must select an authorized device\n' >&2
  exit 2
fi
mkdir -p "$root/build"
workspace="$(mktemp -d "$root/build/platform-$platform_target-XXXXXX")"
printf 'Platform workspace: %s\n' "$workspace"
openapi_kt_build_cli "$workspace/openapi-kt"
cp -R samples/kotlin-platform-smoke "$workspace/project"
if [[ "$platform_target" == android || "$platform_target" == android-device ]]; then
  cp "$workspace/project/android.gradle.kts" "$workspace/project/build.gradle.kts"
  cp "$workspace/project/android-settings.gradle.kts" "$workspace/project/settings.gradle.kts"
  mkdir -p "$workspace/project/src/main"
  printf '<manifest />\n' > "$workspace/project/src/main/AndroidManifest.xml"
fi
gradle_options=()
if [[ "$platform_target" == android-device ]]; then
  probe_package="org.example.openapikt.roadmapprobe.${model_style%_class}"
  gradle_options+=("-PandroidProbePackage=$probe_package")
fi
# Optional installed SDK binary, never an automatic dependency download.
if [[ -n "${ANDROID_AAPT2:-}" ]]; then
  gradle_options+=("-Pandroid.aapt2FromMavenOverride=$ANDROID_AAPT2")
fi
"$workspace/openapi-kt" generate --input tests/fixtures/catalog.json \
  --output "$workspace/project/generated" --package-name org.example.platform --model-style "$model_style"
"$workspace/openapi-kt" generate --input tests/fixtures/catalog.json \
  --output "$workspace/project/generated" --package-name org.example.platform --model-style "$model_style" --check
"$workspace/openapi-kt" generate --input tests/fixtures/platform/model-increments.json \
  --output "$workspace/project/generated-increments" --package-name org.example.platformincrements \
  --model-style "$model_style" --validation --merge-identical-models
"$workspace/openapi-kt" generate --input tests/fixtures/platform/model-increments.json \
  --output "$workspace/project/generated-increments" --package-name org.example.platformincrements \
  --model-style "$model_style" --validation --merge-identical-models --check
"$workspace/openapi-kt" generate --input tests/fixtures/platform/roadmap-increments.json \
  --output "$workspace/project/generated-roadmap" --package-name org.example.platformroadmap \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
"$workspace/openapi-kt" generate --input tests/fixtures/platform/roadmap-increments.json \
  --output "$workspace/project/generated-roadmap" --package-name org.example.platformroadmap \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
"$workspace/openapi-kt" generate --input tests/fixtures/platform/dto-getters.json \
  --output "$workspace/project/generated-getters" --package-name org.example.platformgetters \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
"$workspace/openapi-kt" generate --input tests/fixtures/platform/dto-getters.json \
  --output "$workspace/project/generated-getters" --package-name org.example.platformgetters \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
"$workspace/openapi-kt" generate --input tests/fixtures/platform/enum-field-bounds.json \
  --output "$workspace/project/generated-enum-bounds" --package-name org.example.platformenumbounds \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
"$workspace/openapi-kt" generate --input tests/fixtures/platform/enum-field-bounds.json \
  --output "$workspace/project/generated-enum-bounds" --package-name org.example.platformenumbounds \
  --model-style "$model_style" --validation --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
"${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks \
  -p "$workspace/project" -PplatformTarget="$platform_target" "${gradle_options[@]}" "$task"
if [[ "$platform_target" == android-device ]]; then
  "${PYTHON:-python3}" scripts/kotlin/android_device.py \
    --apk "$workspace/project/build/outputs/apk/androidTest/debug/generated-android-contracts-debug-androidTest.apk" \
    --package "$probe_package"
fi
