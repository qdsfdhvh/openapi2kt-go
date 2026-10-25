#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/split-adoption-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/SplitLockedAdoptionContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import copy,json,sys
p=Path(sys.argv[1]); checked=sys.argv[3]=='on'
d=json.loads(Path('tests/fixtures/http/split-adoption-source.json').read_text())
def bounds(v):
    if isinstance(v,dict):
        if v.get('type')=='string' and 'enum' not in v:v['minLength']=2
        if v.get('type')=='integer':v['minimum']=0
        for child in list(v.values()):bounds(child)
    elif isinstance(v,list):
        for child in v:bounds(child)
if checked:bounds(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
api=copy.deepcopy(d);del api['components']['schemas']['B']
models={'openapi':'3.1.0','components':{'schemas':{'B':d['components']['schemas']['B']}}}
(p/'api.json').write_text(json.dumps(api,indent=2)+'\n');(p/'models.json').write_text(json.dumps(models,indent=2)+'\n')
(p/'split.toml').write_text("config_version=1\n[defaults]\noutput_dir='build/generated'\npackage_name='org.example.split'\nmodel_name_suffix='Dto'\napi_client_name='SplitClient'\nhoist_shared_shapes=true\nlock_hoisted_names=true\nmodel_style='"+sys.argv[2]+"'\nvalidation="+str(checked).lower()+"\n[[specs]]\nname='api'\ninput='api.json'\n[[specs]]\nname='models'\ninput='models.json'\n")
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {splitLockedAdoptionContracts('''+str(checked).lower()+''')}
''')
PY
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.split --model-name-suffix Dto --api-client-name SplitClient --hoist-shared-shapes --lock-hoisted-names --model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  python3 - "$project" <<'PY'
from pathlib import Path
import hashlib,json,sys
p=Path(sys.argv[1]);root=p/'build/generated'
files={str(f.relative_to(root)):{'sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'mode':f.stat().st_mode,'mtime_ns':f.stat().st_mtime_ns} for f in root.rglob('*.kt')}
(p/'before.json').write_text(json.dumps(files))
PY
  action=(--config "$project/split.toml" --adopt-locked-owner --diagnostic-format json)
  if "$workspace/openapi-kt" generate "${action[@]}" --check > "$project/adoption-check.json"; then
   echo 'Expected v3 to v4 metadata difference' >&2; exit 1
  else
   status=$?; [[ "$status" == 3 ]]
  fi
  "$workspace/openapi-kt" validate "${action[@]}" > "$project/adoption-validate.json"
  "$workspace/openapi-kt" generate "${action[@]}" > "$project/adoption-apply.json"
  "$workspace/openapi-kt" generate "${action[@]}" --check > "$project/adoption-replay.json"
  "$workspace/openapi-kt" generate --config "$project/split.toml" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import hashlib,json,sys
p=Path(sys.argv[1]);root=p/'build/generated';before=json.loads((p/'before.json').read_text())
for path,old in before.items():
    f=root/path
    assert {'sha256':hashlib.sha256(f.read_bytes()).hexdigest(),'mode':f.stat().st_mode,'mtime_ns':f.stat().st_mtime_ns}==old,path
m=json.loads((root/'.openapi-kt-manifest.json').read_text())
assert m['version']==4 and set(m['owners'])=={'api','models'}
assert m['owners']['models']==['org/example/split/models/BDto.kt']
assert set(m['owners']['api'])|set(m['owners']['models'])==set(before)
for phase in ['check','validate','apply']:
    report=json.loads((p/('adoption-'+phase+'.json')).read_text())
    assert len(report['locked_owner_adoptions'])==2
PY
  python3 - "$project" "$workspace/openapi-kt" <<'PYEXISTING'
from pathlib import Path
import hashlib,json,subprocess,sys
p=Path(sys.argv[1]);binary=Path(sys.argv[2]);root=p/'build/generated'
def snapshot():
 return {str(f.relative_to(root)):(hashlib.sha256(f.read_bytes()).hexdigest(),f.stat().st_mode,f.stat().st_mtime_ns) for f in root.rglob('*') if f.is_file()}
def invoke(args,code):
 run=subprocess.run([str(binary),*args],capture_output=True,text=True)
 assert run.returncode==code,(run.returncode,run.stdout,run.stderr)
 assert not run.stderr,run.stderr
 return json.loads(run.stdout)
text=(p/'split.toml').read_text()
for donor in ('api','models'):
 owner='consumer_'+donor
 text+="[[specs]]\nname='"+owner+"'\ninput='"+donor+".json'\n"
 cfg=p/'existing.toml';cfg.write_text(text)
 flags=['--config',str(cfg),'--spec='+owner,'--adopt-locked-owner','--adopt-locked-owner-from='+donor,'--diagnostic-format=json']
 before=snapshot();manifest=json.loads((root/'.openapi-kt-manifest.json').read_text())
 check=invoke(['generate',*flags,'--check'],3);validated=invoke(['validate',*flags],0)
 assert check['locked_owner_adoptions']==validated['locked_owner_adoptions']
 assert check['locked_owner_adoptions'][0]['from_owner']==donor
 assert snapshot()==before
 applied=invoke(['generate',*flags],0)
 assert check['locked_owner_adoptions']==applied['locked_owner_adoptions']
 after=snapshot()
 for path,old in before.items():
  if path!='.openapi-kt-manifest.json':assert after[path]==old,path
 next=json.loads((root/'.openapi-kt-manifest.json').read_text())
 assert next['owners'][owner]==manifest['owners'][donor]
 assert all(next['owners'][name]==paths for name,paths in manifest['owners'].items())
 assert all(next['name_locks'][name]==hash for name,hash in manifest['name_locks'].items())
 assert next['name_locks'][owner]==manifest['name_locks'][donor]
 for command in (['generate',*flags,'--check'],['validate',*flags],['generate',*flags]):invoke(command,0)
 assert snapshot()==after
 for phase,value in [('check',check),('validate',validated),('apply',applied)]:
  (p/('existing-'+donor+'-'+phase+'.json')).write_text(json.dumps(value,indent=2)+'\n')
invoke(['generate','--config',str(cfg),'--check','--diagnostic-format=json'],0)
print('Existing-v4 donor adoption preserved all compiled Kotlin, memberships, raw locks and attributes')
PYEXISTING
  env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
   OPENAPI_KT_PROJECTION_OUTPUT="$project/build/generated-options" \
   OPENAPI_KT_PROJECTION_INPUT="$project/input.json" \
   OPENAPI_KT_PROJECTION_STYLE="$style" OPENAPI_KT_PROJECTION_CHECKED="$checked" \
   "${GO:-go}" test ./core -run '^TestExportNameLockProjection$' -count=1
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runSplitLockedAdoptionContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
   echo "PREPARED SPLIT LOCKED ADOPTION $style $checked: $project"
   continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runSplitLockedAdoptionContracts
 done
done
echo "Fresh split-adoption projects: $workspace"
