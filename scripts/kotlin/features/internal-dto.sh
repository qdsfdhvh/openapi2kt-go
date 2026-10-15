#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/internal-dto-smoke-XXXXXX")"
fixture="${INTERNAL_DTO_FIXTURE:-tests/fixtures/http/schema-internal-dtos.json}"
openapi_kt_build_cli "$workspace/openapi-kt"
for profile in data-off data-on plain-off plain-on all-on; do
  style=data_class; checked=off
  case "$profile" in plain-*) style=plain_class;; esac
  case "$profile" in *-on) checked=on;; esac
  project="$workspace/$profile"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/SchemaInternalDTOContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" "$style" "$fixture" "${INTERNAL_DTO_ROOT_ALIASES:-0}" "${INTERNAL_DTO_NAMED_TERMINALS:-0}" "${INTERNAL_DTO_ALIAS_TERMINALS:-0}" "${INTERNAL_DTO_ROOT_BRIDGES:-0}" <<'PY'
import json, shutil, sys
from pathlib import Path
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'; style=sys.argv[3]
doc=json.loads(Path(sys.argv[4]).read_text())
named=sys.argv[6]=='1'
if checked:
    target=doc['components']['schemas']['Node'] if named else doc['components']['schemas']['Tree/key~']['properties']['node']
    target['properties']['id']['minLength']=2
if named:
    # Adapt only the test consumer to the new physical named classifiers.
    # The generated Kotlin is never rewritten; all existing wire assertions run.
    consumer=p/'src/main/kotlin/org/example/contracts/SchemaInternalDTOContracts.kt'
    text=consumer.read_text().replace('import org.example.internaldtos.models.TreeKeyDto', 'import org.example.internaldtos.models.NodeDto\nimport org.example.internaldtos.models.MaybeDto')
    consumer.write_text(text.replace('TreeKeyDto.Node', 'NodeDto').replace('TreeKeyDto.Maybe', 'MaybeDto'))
(p/'input.json').write_text(json.dumps(doc))
ref={'$ref':'#/components/schemas/Owner/properties/node'}
value=ref
for _ in range(46): value={'type':'array','items':value}
probes={'openapi':'3.1.0','components':{'schemas':{
    'Owner':{'properties':{'node':{'properties':{'id':{'type':'string'}},'required':['id']}}},
    'Wide':{'properties':{f'p{i}':ref for i in range(240)}},
    'Deep':{'properties':{'value':value}}
}}}
if 'ChainRows' in doc['components']['schemas']:
    owner=probes['components']['schemas']['Owner']['properties']
    owner['shortHop']={'$ref':'#/components/schemas/Owner/properties/node'}
    for i in range(46):
        owner[f'hop{i}']={'$ref':f'#/components/schemas/Owner/properties/'+(f'hop{i+1}' if i<45 else 'node')}
    probes['components']['schemas']['Wide']['properties']={f'p{i}':{'$ref':'#/components/schemas/Owner/properties/shortHop'} for i in range(240)}
    probes['components']['schemas']['Deep']['properties']['value']={'$ref':'#/components/schemas/Owner/properties/hop0'}
if named:
    schemas=probes['components']['schemas']
    schemas['Node']={'properties':{'id':{'type':'string'}},'required':['id']}
    owner=schemas['Owner']['properties']
    owner['node']={'$ref':'#/components/schemas/Node'}
    for i in range(45):
        owner[f'hop{i}']={'$ref':'#/components/schemas/Owner/properties/'+(f'hop{i+1}' if i<44 else 'node')}
    schemas['Deep']['properties']['value']={'$ref':'#/components/schemas/Owner/properties/hop0'}
aliased=sys.argv[7]=='1'
if aliased:
    schemas=probes['components']['schemas']
    physical='Node' if named else 'Owner/properties/node'
    schemas['DTOAlias']={'$ref':'#/components/schemas/'+physical,'nullable':True}
    owner=schemas['Owner']['properties']
    owner['aliasTerminal']={'$ref':'#/components/schemas/DTOAlias','nullable':False}
    for i in range(44):
        owner[f'aliasHop{i}']={'$ref':'#/components/schemas/Owner/properties/'+(f'aliasHop{i+1}' if i<43 else 'aliasTerminal')}
    schemas['Wide']['properties']={f'p{i}':{'$ref':'#/components/schemas/Owner/properties/aliasTerminal'} for i in range(240)}
    schemas['Deep']['properties']['value']={'$ref':'#/components/schemas/Owner/properties/aliasHop0'}
    alias_consumer=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOAliasChainContracts.kt').read_text()
    (p/'src/main/kotlin/org/example/contracts/DTOAliasChainContracts.kt').write_text(alias_consumer.replace('org.example.dtoaliaschains.models.', 'org.example.internaldtos.models.'))
bridges=sys.argv[8]=='1'
if bridges:
    schemas=probes['components']['schemas']; owner=schemas['Owner']['properties']
    schemas['LegacyDTOAlias']=schemas['DTOAlias']
    owner['bridgeAlias']={'$ref':'#/components/schemas/LegacyDTOAlias'}
    schemas['DTOAlias']={'$ref':'#/components/schemas/Owner/properties/bridgeAlias'}
    for i in range(44): del owner[f'aliasHop{i}']
    for i in range(42):
        owner[f'aliasHop{i}']={'$ref':'#/components/schemas/Owner/properties/'+(f'aliasHop{i+1}' if i<41 else 'aliasTerminal')}
    text=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootDTOBridgeContracts.kt').read_text()
    (p/'src/main/kotlin/org/example/contracts/RootDTOBridgeContracts.kt').write_text(text.replace('org.example.rootdtobridges.models.', 'org.example.internaldtos.models.'))
roots=sys.argv[5]=='1'
if roots:
    schemas=probes['components']['schemas']
    schemas['NodeAlias']=ref
    schemas['Wide']['properties']={f'p{i}':{'$ref':'#/components/schemas/NodeAlias'} for i in range(240)}
    for i in range(46):
        schemas[f'A{i:02d}']={'$ref':'#/components/schemas/'+('Owner/properties/node' if i==0 else f'A{i-1:02d}')}
    schemas['Deep']['properties']['value']={'$ref':'#/components/schemas/A45'}
    shutil.copyfile('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootInternalDTOAliasContracts.kt',p/'src/main/kotlin/org/example/contracts/RootInternalDTOAliasContracts.kt')
(p/'limits.json').write_text(json.dumps(probes))
copy='' if style=='plain_class' else '''
    val original=org.example.internaldtos.models.TreeKeyDto.Node("valid")
    check(original.copy(id="copy").id=="copy")
'''
if named:
    copy=copy.replace('org.example.internaldtos.models.TreeKeyDto.Node', 'org.example.internaldtos.models.NodeDto')
if roots and style!='plain_class':
    copy+='''    check(org.example.rootinternaldtos.models.NodeAliasDto("valid").copy(id="alias-copy").id=="alias-copy")\n'''
if aliased and style!='plain_class':
    copy+='''    check(org.example.internaldtos.models.NodeAliasChainDto("valid").copy(id="alias-copy").id=="alias-copy")\n'''
extra=('    rootInternalDTOAliasContracts('+str(checked).lower()+')\n') if roots else ''
if aliased:
    extra+='    dtoAliasChainContracts()\n'
if bridges:
    extra+='    rootDTOBridgeContracts()\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
'''+copy+extra+'''    schemaInternalDTOContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runInternalDtoContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  flags=(--model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$profile" == all-on ]]; then
    flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models)
  fi
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.internaldtos --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.internaldtos --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.internaldtolimits --model-name-suffix Dto "${flags[@]}"
  if [[ "${INTERNAL_DTO_ROOT_ALIASES:-0}" == 1 ]]; then
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-root-internal" --package-name org.example.rootinternaldtos --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-root-internal" --package-name org.example.rootinternaldtos --model-name-suffix Dto "${flags[@]}" --check
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runInternalDtoContracts
  echo "FRESH INTERNAL DTO PROFILE $profile PASS: $project"
done
echo "Fresh internal DTO projects: $workspace"
