#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
if [[ $# -lt 1 || $# -gt 2 ]]; then
  printf 'Usage: bash scripts/kotlin/platforms.sh {jvm|metadata|js|native|android} [data_class|plain_class]\n' >&2
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
  *) printf 'Unsupported target\n' >&2; exit 2 ;;
esac
case "$model_style" in data_class|plain_class) ;; *) printf 'Unsupported model style\n' >&2; exit 2 ;; esac
mkdir -p "$root/build"
workspace="$(mktemp -d "$root/build/platform-$platform_target-XXXXXX")"
printf 'Platform workspace: %s\n' "$workspace"
openapi_kt_build_cli "$workspace/openapi-kt"
cp -R samples/kotlin-platform-smoke "$workspace/project"
if [[ "$platform_target" == android ]]; then
  cp "$workspace/project/android.gradle.kts" "$workspace/project/build.gradle.kts"
  cp "$workspace/project/android-settings.gradle.kts" "$workspace/project/settings.gradle.kts"
  mkdir -p "$workspace/project/src/main"
  printf '<manifest />\n' > "$workspace/project/src/main/AndroidManifest.xml"
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
"${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks \
  -p "$workspace/project" -PplatformTarget="$platform_target" "$task"
