#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/literal-false-collection-fields-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseCollectionFieldsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  validation=(); [[ "$checked" == off ]] || validation+=(--validation)
  for mode in base optimized; do
   package=org.example.literalcollections;output="$project/build/generated";suffix=;options=()
   if [[ "$mode" == optimized ]]; then
    package+=optimized;output="$project/build/generated-limits";suffix=Dto
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   fi
   args=(--input tests/fixtures/http/literal-false-collection-fields.json --output "$output" --package-name "$package" --api-client-name LiteralCollectionsClient --model-name-suffix "$suffix" --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
shallow={'type':'array','items':False}
if checked:shallow['maxItems']=0
props={f'p{i:03d}':shallow for i in range(239)}
leaf=False
for _ in range(15):
 layer={'type':'array','items':leaf}
 if checked:layer['maxItems']=1
 leaf=layer
props['p239']=leaf
required={'required':list(props),'properties':props}
if checked:required['minProperties']=240
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':props},'WideRequired':required}}}))
PYWIDE
  args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.literalcollectionswide --model-name-suffix Dto --model-style "$style")
  "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}"
  "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" --check
  corpus_args=(--input tests/fixtures/p1/invalid/array-items-boolean.json --output "$project/build/generated-corpus" --package-name org.example.literalcollectionscorpus --model-name-suffix "" --model-style "$style")
  "$workspace/openapi-kt" generate "${corpus_args[@]}" "${validation[@]}"
  "$workspace/openapi-kt" generate "${corpus_args[@]}" "${validation[@]}" --check
  python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=str(sys.argv[2]=='on').lower();consumer=(p/'src/main/kotlin/org/example/contracts/LiteralFalseCollectionFieldsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.literalcollections.LiteralCollectionsClient','org.example.literalcollectionsoptimized.LiteralCollectionsClient')
for name in json.loads(Path('tests/fixtures/http/literal-false-collection-fields.json').read_text())['components']['schemas']:
 optimized=optimized.replace('org.example.literalcollections.models.'+name+' as ','org.example.literalcollectionsoptimized.models.'+name+'Dto as ')
(opt/'LiteralFalseCollectionFieldsContracts.kt').write_text(optimized)
assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries'],'unrelated ordinary request shapes retain nonempty locks'
for name in ['WideDto','WideRequiredDto']:
 assert 'AbsentSerializer' not in (p/'build/generated-options/org/example/literalcollectionswide/models'/(name+'.kt')).read_text()
deep='emptyList<org.example.literalcollectionswide.models.WideDto.FalseValue>()'
for _ in range(14):deep='listOf('+deep+')'
wide=r'''
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideLiteralFalseCollections() {
 val value=org.example.literalcollectionswide.models.WideDto(p000=emptyList(),p239=DEEP)
 check(value.p000!!.isEmpty())
 val optional=org.example.literalcollectionswide.models.WideDto.serializer().descriptor
 val required=org.example.literalcollectionswide.models.WideRequiredDto.serializer().descriptor
 check(optional.elementsCount==240 && required.elementsCount==240)
 for(index in 0 until 240) {check(optional.isElementOptional(index));check(!required.isElementOptional(index))}
 val all=requiredWide()
 check(all.p000.isEmpty() && all.p239.isEmpty())
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=kotlinx.serialization.json.Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(codec.decodeFromString<org.example.literalcollectionswide.models.WideDto>(codec.encodeToString(value)).p000!!.isEmpty())
  check(codec.decodeFromString<org.example.literalcollectionswide.models.WideRequiredDto>(codec.encodeToString(all)).p239.isEmpty())
  check(runCatching {codec.decodeFromString<org.example.literalcollectionswide.models.WideDto>("{\"p000\":[null]}")}.isFailure)
  check(runCatching {codec.decodeFromString<org.example.literalcollectionswide.models.WideDto>("{\"p239\":[[],[]]}")}.isFailure == CHECKED)
  val corpus=org.example.literalcollectionscorpus.models.Record(value=emptyList())
  check(codec.encodeToString(corpus)=="{\"value\":[]}")
  check(runCatching {codec.decodeFromString<org.example.literalcollectionscorpus.models.Record>("{\"value\":[null]}")}.isFailure)
 }
}
'''.replace('DEEP',deep).replace('CHECKED',checked)
construct='private fun requiredWide() = org.example.literalcollectionswide.models.WideRequiredDto('+','.join(['emptyList()']*240)+')\n'
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.encodeToString\nimport kotlinx.serialization.decodeFromString\n'+construct+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main()=runBlocking {wideLiteralFalseCollections();literalFalseCollectionFieldsContracts();org.example.optimizedcontracts.literalFalseCollectionFieldsContracts()}\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main {kotlin.srcDir(layout.buildDirectory.dir("generated-corpus"))}
tasks.register<JavaExec>("runLiteralFalseCollectionFieldsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
   echo "PREPARED LITERAL FALSE COLLECTION FIELDS $style $checked: $project"
   continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runLiteralFalseCollectionFieldsContracts
  echo "FRESH LITERAL FALSE COLLECTION FIELDS PROFILE $style-$checked PASS: $project"
 done
done
