@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import org.example.dedup.DeduplicateContractsApiClient
import org.example.dedup.api.models.AlphaResponseDto
import org.example.dedup.api.models.BetaResponseDto
import org.example.dedup.api.models.BetaInputDto
import org.example.dedup.api.models.BetaModeDto
import org.example.dedup.api.models.GuardResponseDto
import org.example.dedup.models.Patch
import org.example.dedupplain.DeduplicateContractsApiClient as PlainClient
import org.example.dedupplain.api.models.AlphaResponseDto as PlainAlpha
import org.example.dedupplain.api.models.BetaResponseDto as PlainBeta
import org.example.dedupplain.api.models.BetaInputDto as PlainInput
import org.example.dedupplain.api.models.BetaModeDto as PlainMode
import org.example.dedupplain.models.Patch as PlainPatch

private fun body(value:OutgoingContent):String=when(value){is OutgoingContent.ContentWrapper->body(value.delegate());is OutgoingContent.ByteArrayContent->value.bytes().decodeToString();else->error("Unexpected body")}
private fun rejects(block:()->Unit){try{block();error("invalid input accepted")}catch(cause:Throwable){check(generateSequence(cause){it.cause}.any{it is IllegalArgumentException||it is SerializationException})}}
suspend fun deduplicateModelContracts(){
 for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
 val json=Json{encodeDefaults=defaults;explicitNulls=nulls;ignoreUnknownKeys=true}
 val root=AlphaResponseDto("ok",AlphaResponseDto.Child(AlphaResponseDto.Child.Status.fromSerialName("future")))
 val alias=BetaResponseDto("ok",root.child)
 check(root==alias&&alias.copy(id="new").component1()=="new")
 check(json.decodeFromJsonElement(BetaResponseDto.serializer(),json.encodeToJsonElement(alias))==alias)
 check(BetaResponseDto.serializer().descriptor.serialName==AlphaResponseDto.serializer().descriptor.serialName)
 check(BetaResponseDto::class.java===AlphaResponseDto::class.java)
 rejects{GuardResponseDto("ok",GuardResponseDto.Child(GuardResponseDto.Child.Status.READY))}
 rejects{BetaResponseDto("x",root.child)}
 val plain=PlainBeta("ok",PlainAlpha.Child(PlainAlpha.Child.Status.fromSerialName("future")))
 check(plain==PlainAlpha("ok",plain.child));check(plain.toString()=="AlphaResponseDto");check(PlainBeta::class.java.methods.none{it.name=="copy"||it.name.startsWith("component")})
 check(json.encodeToJsonElement(root)==json.encodeToJsonElement(plain))
 check(json.decodeFromJsonElement(PlainBeta.serializer(),json.encodeToJsonElement(plain))==plain)
 for(codec in listOf(BetaResponseDto.serializer(),AlphaResponseDto.serializer())){rejects{json.decodeFromString(codec,"{\"id\":\"ok\",\"child\":{\"status\":\"future\"},\"kind\":\"record\",\"extra\":1,\"other\":2}")}}
 val response=json.encodeToJsonElement(root).toString()
 var expected=json.parseToJsonElement("{}");var count=0
 val engine=MockEngine{request->count++;check(request.url.encodedPath=="/beta"&&request.url.parameters["mode"]=="fast");check(json.parseToJsonElement(body(request.body))==expected);respond(response,HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
 val http=HttpClient(engine){install(ContentNegotiation){json(json)}}
 try{
 val api=DeduplicateContractsApiClient(http,"http://example.test/").default
 val pApi=PlainClient(http,"http://example.test/").default
 for(state in listOf<Patch<String>>(Patch.Absent,Patch.Null,Patch.Value("note"))){val input=BetaInputDto("ok",state);expected=json.encodeToJsonElement(input);check(api.beta(mode=BetaModeDto.fromSerialName("fast"),input=input)==root)}
 for(state in listOf<PlainPatch<String>>(PlainPatch.Absent,PlainPatch.Null,PlainPatch.Value("note"))){val input=PlainInput("ok",state);expected=json.encodeToJsonElement(input);check(pApi.beta(mode=PlainMode.fromSerialName("fast"),input=input)==plain)}
 check(count==6)
 }finally{http.close();engine.close()}
 println("DeduplicateProduct Json defaults=$defaults nulls=$nulls data-plain-alias-canonical-nested-identity-descriptor-copy-checks-enum-Patch-count-typedHTTP checks passed")
 }
}
