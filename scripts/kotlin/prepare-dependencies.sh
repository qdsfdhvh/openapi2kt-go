#!/usr/bin/env bash
# Explicit development/CI dependency acquisition. Generation/tests remain offline.
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/kotlin-dependencies-XXXXXX")"
mkdir -p "$workspace/src/main/kotlin/org/example/contracts"
cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$workspace/"
cp scripts/kotlin/DependencyCacheProbe.kt "$workspace/src/main/kotlin/org/example/contracts/"
python3 - "$workspace/build.gradle.kts" <<'PY'
from pathlib import Path
import sys
build = Path(sys.argv[1])
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runDependencyCacheProbe") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.DependencyCacheProbeKt")
}
''')
PY
"${GRADLE:-gradle}" --no-daemon --no-build-cache -p "$workspace" runDependencyCacheProbe "$@"
echo "Kotlin dependency preparation passed: $workspace"
