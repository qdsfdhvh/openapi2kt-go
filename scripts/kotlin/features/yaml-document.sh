#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/yaml-document-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/yaml-document/YAMLDocumentContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--package-name org.example.yaml --api-client-name YamlClient --model-name-suffix Dto --model-style "$style" --diagnostic-format=json)
    if [[ "$checked" == true ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/yaml/records.json --output "$project/control" "${flags[@]}" > "$project/json-report.json"
    "$workspace/openapi-kt" generate --input tests/fixtures/yaml/records.yaml --output "$project/build/generated" "${flags[@]}" > "$project/yaml-report.json"
    "$workspace/openapi-kt" generate --input tests/fixtures/yaml/records.yaml --output "$project/build/generated" "${flags[@]}" --check > "$project/check-report.json"
    python3 - "$project" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1])
reports = [json.loads((p/n).read_text()) for n in ('json-report.json', 'yaml-report.json', 'check-report.json')]
assert reports[0] == reports[1] == reports[2], 'JSON/YAML/check report mismatch'
def files(root):
    return {f.relative_to(root).as_posix(): f.read_bytes() for f in root.rglob('*') if f.is_file()}
assert files(p/'control') == files(p/'build/generated'), 'JSON/YAML sources or manifests differ'
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runYAMLDocumentContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.YAMLDocumentContractsKt")
}
''')
PY
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runYAMLDocumentContracts
    printf 'Fresh YAML document style=%s validation=%s PASS: %s\n' "$style" "$checked" "$project"
  done
done
