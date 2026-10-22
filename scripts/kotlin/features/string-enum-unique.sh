#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/string-enum-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[3]=='on'
d=json.loads(Path('tests/fixtures/http/string-enum-unique.json').read_text())
def control(v):
    if isinstance(v,dict):
        v.pop('uniqueItems',None)
        v.pop('minLength',None)
        for x in v.values():control(x)
    elif isinstance(v,list):
        for x in v:control(x)
if not checked:control(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
copy=''
if sys.argv[2]=='data_class':
    copy='''val original = HolderDto()
    val copied = runCatching { original.copy(codes = listOf(HolderDto.CodesInner.READY, HolderDto.CodesInner.Unrecognized("ready"))) }
    check(copied.isFailure == '''+str(checked).lower()+''')
    val refCopied = runCatching { original.copy(refCodes = listOf(org.example.enumunique.models.StateDto.READY, org.example.enumunique.models.StateDto.Unrecognized("ready"))) }
    check(refCopied.isFailure == '''+str(checked).lower()+''')'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import org.example.enumunique.models.HolderDto
fun main() = runBlocking {
    '''+copy+'''
    stringEnumUniqueContracts('''+str(checked).lower()+''')
}
''')
PY
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumunique --model-name-suffix Dto --model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  python3 - "$project" <<'PYOWNER'
from pathlib import Path
import re,sys
p=Path(sys.argv[1])
source=(p/'build/generated/org/example/enumunique/models/StateDto.kt').read_text()
match=re.search(r'public typealias StateDto = (\S+)',source)
if match:
    consumer=p/'src/main/kotlin/org/example/contracts/StringEnumUniqueContracts.kt'
    consumer.write_text(consumer.read_text().replace('import org.example.enumunique.models.StateDto as RefCode','import '+match.group(1)+' as RefCode'))
PYOWNER
  if [[ "$checked" == off ]]; then
    if "$workspace/openapi-kt" validate --input tests/fixtures/http/string-enum-unique.json > "$project/rejected.log" 2>&1; then
      echo 'Validation-off accepted String enum uniqueItems' >&2; exit 1
    fi
    rg -q E_SCHEMA_UNSUPPORTED "$project/rejected.log"
  fi
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runStringEnumUniqueContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
    echo "PREPARED STRING ENUM UNIQUE $style $checked: $project"
    continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runStringEnumUniqueContracts
 done
done
echo "Fresh String enum uniqueness projects: $workspace"
