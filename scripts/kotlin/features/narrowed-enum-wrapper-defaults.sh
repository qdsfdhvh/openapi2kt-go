#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/narrowed-enum-wrapper-defaults-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in false true; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumNullMemberContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='true';ref='#/components/schemas/'
d=json.loads(Path('tests/fixtures/http/string-enum-null-members.json').read_text())
s=d['components']['schemas']
s['State']['default']=None;s['Alias']['default']=None;s['False']['default']='ready'
s['Again']={'$ref':ref+'False','nullable':True,'default':None}
s['Bare']={'$ref':ref+'False'}
native=''
native_imports=''
if checked:
 for name,kind,fmt,values,value in [
  ('Flag','boolean',None,[True,None,False],True),
  ('Count','integer',None,[1,None,2],2),
  ('Big','integer','int64',[9223372036854775807,None,0],9223372036854775807),
  ('Ratio','number','float',[0.1,None,1.0],0.1),
  ('Double','number',None,[0.1,None,1.0],0.1),
 ]:
  s[name]={'type':kind,'nullable':True,'enum':values,'default':None}
  if fmt:s[name]['format']=fmt
  if name=='Count':s[name]['minimum']=2
  s[name+'Narrow']={'$ref':ref+name,'nullable':False,'default':value}
 body={'$ref':ref+'CountNarrow'}
 d['paths']['/count']={'post':{'operationId':'echoCount','requestBody':{'required':True,'content':{'application/json':{'schema':body}}},'responses':{'200':{'content':{'application/json':{'schema':body}}}}}}
 d['paths']['/watch']['get']['responses']['200']['content']['text/event-stream']['schema']['oneOf'].append({'properties':{'event':{'const':'count'},'data':body},'required':['data']})
 native_imports='''
import org.example.stringenumnullmembers.models.CountNarrowDto
import org.example.stringenumnullmembers.models.`FlagNarrowDto default`
import org.example.stringenumnullmembers.models.`CountNarrowDto default`
import org.example.stringenumnullmembers.models.`BigNarrowDto default`
import org.example.stringenumnullmembers.models.`RatioNarrowDto default`
import org.example.stringenumnullmembers.models.`DoubleNarrowDto default`
'''
 native='''
 check(`FlagNarrowDto default`())
 check(`CountNarrowDto default`()==2)
 check(`BigNarrowDto default`()==Long.MAX_VALUE)
 check(`RatioNarrowDto default`().toRawBits()==0.1f.toRawBits())
 check(`DoubleNarrowDto default`()==0.1)
 check(!serializer<CountNarrowDto>().descriptor.isNullable)
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  var response="2"
  var stream="event: count\\ndata: 2\\n\\n"
  var calls=0
  val engine=WrapperDefaultEngine(MockEngineConfig().apply {addHandler {request ->
   when(request.url.encodedPath) {
    "/count" -> {calls++;check(wrapperBody(request.body)=="2");respond(response,headers=headersOf(HttpHeaders.ContentType,"application/json"))}
    "/watch" -> respond(stream,headers=headersOf(HttpHeaders.ContentType,"text/event-stream"))
    else -> error("Unexpected synthetic wrapper route")
   }
  }})
  val http=HttpClient(engine) {install(ContentNegotiation){json(codec)};install(SSE)}
  try {
   val api=DefaultApi(http,"https://example.test/",codec)
   check(api.echoCount(`CountNarrowDto default`())==2)
   val before=calls
   check(runCatching {api.echoCount(1)}.isFailure && calls==before)
   response="1";check(runCatching {api.echoCount(2)}.isFailure)
   response="2";check(api.echoCount(2)==2)
   check((api.watchState().toList().single() as WatchStateEventDto.Count).data==2)
   stream="event: count\\ndata: 1\\n\\n";check(runCatching {api.watchState().toList()}.isFailure)
   stream="event: count\\ndata: 2\\n\\n";check((api.watchState().toList().single() as WatchStateEventDto.Count).data==2)
  } finally {http.close()}
 }
'''
(p/'input.json').write_text(json.dumps(d)+'\n')
leaf={'enum':['on',None,'off']}
resources={'State':leaf,'Narrow':{'$ref':ref+'State','nullable':False,'default':'on'},'Full':{'enum':[None]+[f'v{i}' for i in range(239)]},'FullNarrow':{'$ref':ref+'Full','nullable':False,'default':'v238'},'Large':{'enum':[' '*32766,None]},'LargeNarrow':{'$ref':ref+'Large','nullable':False,'default':' '*32766}}
resources['Wide']={'properties':{f'p{i:03d}':{'$ref':ref+'Narrow'} for i in range(240)},'required':[f'p{i:03d}' for i in range(240)]}
for i in range(48):
 resources[f'Alias{i}']={'$ref':ref+('State' if i==0 else f'Alias{i-1}')}
resources['Alias47'].update(nullable=False,default='on')
(p/'resources.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':resources}})+'\n')
shape={'properties':{'value':{'$ref':ref+'False'}}}
paths={f'/{name}':{'get':{'operationId':name,'responses':{'200':{'content':{'application/json':{'schema':shape}}}}}} for name in ['alpha','beta']}
(p/'options.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths})+'\n')
main='''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class,io.ktor.utils.io.InternalAPI::class)
package org.example.contracts
import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.client.plugins.sse.SSECapability
import io.ktor.client.request.HttpRequestData
import io.ktor.client.request.HttpResponseData
import io.ktor.client.request.ResponseAdapterAttributeKey
import io.ktor.http.HttpHeaders
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteReadChannel
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import org.example.stringenumnullmembers.models.StateDto
import org.example.stringenumnullmembers.models.BareDto
import org.example.stringenumnullmembers.models.`StateDto default`
import org.example.stringenumnullmembers.models.`AliasDto default`
import org.example.stringenumnullmembers.models.`FalseDto default`
import org.example.stringenumnullmembers.models.`AgainDto default`
import org.example.stringenumnullmembers.api.DefaultApi
import org.example.stringenumnullmembers.api.models.WatchStateEventDto
'''+native_imports+'''
private class WrapperDefaultEngine(config:MockEngineConfig):MockEngine(config) {
 override val supportedCapabilities=super.supportedCapabilities+SSECapability
 override suspend fun execute(data:HttpRequestData):HttpResponseData {
  val response=super.execute(data)
  val adapter=data.attributes.getOrNull(ResponseAdapterAttributeKey)?:return response
  val body=adapter.adapt(data,response.statusCode,response.headers,response.body as ByteReadChannel,data.body,response.callContext)?:return response
  return HttpResponseData(response.statusCode,response.requestTime,response.headers,response.version,body,response.callContext)
 }
}
private fun wrapperBody(body:OutgoingContent):String=when(body) {
 is OutgoingContent.ContentWrapper -> wrapperBody(body.delegate())
 is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
 else -> error("Unexpected synthetic wrapper body")
}
fun main()=runBlocking {
 stringEnumNullMemberContracts()
 check(`StateDto default`()==null && `AliasDto default`()==null)
 check(`FalseDto default`()===StateDto.READY && `AgainDto default`()==null)
 check(!serializer<BareDto>().descriptor.isNullable)
 check(Json.decodeFromString<BareDto>("\\"future\\"")==StateDto.Unrecognized("future"))
 check(runCatching {Json.decodeFromString<BareDto>("null")}.isFailure)
 check(org.example.wrapperoptions.api.models.AlphaResponseDto(value=org.example.wrapperoptions.models.StateDto.READY).value?.serialName=="ready")
 check(org.example.wrapperoptions.api.models.BetaResponseDto.serializer().descriptor.serialName==org.example.wrapperoptions.models.ResponseDto.serializer().descriptor.serialName)
 check(org.example.wrapperresources.models.`FullNarrowDto default`().serialName=="v238")
 check(org.example.wrapperresources.models.`LargeNarrowDto default`().serialName.length==32766)
 check(org.example.wrapperresources.models.`Alias47Dto default`()===org.example.wrapperresources.models.StateDto.ON)
 val wide=org.example.wrapperresources.models.WideDto('''+','.join(['org.example.wrapperresources.models.StateDto.ON']*240)+''')
 check(org.example.wrapperresources.models.WideDto.serializer().descriptor.elementsCount==240 && wide.p239.serialName=="on")
'''+native+'''}
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(main)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNarrowedEnumWrapperDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  validation=(); [[ "$checked" == false ]] || validation+=(--validation)
  for entry in input:generated:org.example.stringenumnullmembers resources:generated-options:org.example.wrapperresources options:generated-limits:org.example.wrapperoptions; do
   IFS=: read -r input output package <<< "$entry"
   options=()
   [[ "$input" != options ]] || options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   args=(--input "$project/$input.json" --output "$project/build/$output" --package-name "$package" --model-name-suffix Dto --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries']
assert 'BareDto default' not in (p/'build/generated/org/example/stringenumnullmembers/models/BareDto.kt').read_text()
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNarrowedEnumWrapperDefaultsContracts
  python3 - "$project" "$workspace/openapi-kt" "$style" "$checked" <<'PY'
from pathlib import Path
import json,struct,subprocess,sys
p=Path(sys.argv[1]);binary,style,checked=sys.argv[2:]
# Inspect the actual this_class constant after compilation. The short member
# name is only a lower bound; its original token still has 32766 bytes.
path=p/'build/classes/kotlin/main/org/example/wrapperresources/models/LargeDto$VALUE.class'
data=path.read_bytes();assert data[:4]==b'\xca\xfe\xba\xbe'
count=struct.unpack_from('>H',data,8)[0];offset=10;pool={};i=1
while i<count:
 tag=data[offset];offset+=1
 if tag==1:
  n=struct.unpack_from('>H',data,offset)[0];offset+=2
  pool[i]=(tag,data[offset:offset+n]);offset+=n
 elif tag in (7,8,16,19,20):
  pool[i]=(tag,struct.unpack_from('>H',data,offset)[0]);offset+=2
 elif tag in (3,4,9,10,11,12,17,18):offset+=4
 elif tag in (5,6):offset+=8;i+=1
 elif tag==15:offset+=3
 else:raise AssertionError(('Unexpected classfile constant',tag))
 i+=1
this_class=struct.unpack_from('>H',data,offset+2)[0]
name=pool[pool[this_class][1]][1]
assert name==b'org/example/wrapperresources/models/LargeDto$VALUE'
assert len(b'VALUE')<len(name)<=65535
# Preserve the exact originally uncompilable slash-token case as a native
# refusal, against the same producer used for this fresh Kotlin project.
d=json.loads((p/'resources.json').read_text())
d['components']['schemas']['Large']['enum']=['/'*32766,None]
d['components']['schemas']['LargeNarrow']['default']='/'*32766
bad=p/'overflow.json';bad.write_text(json.dumps(d)+'\n')
output=p/'overflow-output'
args=[binary,'generate','--input',str(bad),'--output',str(output),'--package-name','org.example.enumlimit','--model-name-suffix','Dto','--model-style',style,'--diagnostic-format','json','--check']
if checked=='true':args.append('--validation')
result=subprocess.run(args,capture_output=True,text=True)
report=json.loads(result.stdout)
assert result.returncode==1 and not result.stderr and not output.exists()
assert report['diagnostics'][0]['code']=='E_MODEL_LIMIT'
assert report['diagnostics'][0]['pointer']=='/components/schemas/Large/enum/0'
print('Actual enum class-name lower bound and original overflow refusal PASS:',name.decode())
PY
  echo "FRESH NARROWED ENUM WRAPPER PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh narrowed enum wrapper projects: $workspace"
