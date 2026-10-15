@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rootrawaliases.RootRawAliasApiClient
import org.example.rootrawaliases.api.models.SubmitInputDto
import org.example.rootrawaliases.api.models.WatchEventDto
import org.example.rootrawaliases.models.ARawDto
import org.example.rootrawaliases.models.AOtherDto
import org.example.rootrawaliases.models.HolderDto
import org.example.rootrawaliases.models.Patch
import org.example.rootrawprobes.models.ANearDto as RawNearDto
import org.example.rootrawprobes.models.AWideDto as RawWideDto

private fun rawAliasBody(body: OutgoingContent): String = when(body){
    is OutgoingContent.ContentWrapper -> rawAliasBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected raw alias body")
}
private suspend fun rawAliasReject(block: suspend () -> Unit) {
    try {block();error("Malformed raw JSON was accepted")}
    catch (cause: Throwable){check(generateSequence(cause){it.cause}.any {it is SerializationException}) {"Wrong raw alias failure $cause"}}
}

suspend fun rootRawAliasContracts() {
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)){
        val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
        val values=listOf("{}","[]","[null,{\"future\":true}]","\"text\"","true","-9223372036854775808","1.25","null").map(codec::parseToJsonElement)
        for(value in values){
            val raw: ARawDto=value
            val other: AOtherDto=raw
            val serializer=JsonElement.serializer()
            check(codec.decodeFromString(serializer,codec.encodeToString(serializer,other))==raw)
            val holder=HolderDto(raw,other,listOf(raw),mapOf("value" to other))
            val encoded=codec.encodeToString(holder)
            check(codec.decodeFromString<HolderDto>(encoded)==holder)
            check((codec.parseToJsonElement(encoded) as JsonObject)["raw"]==raw)
            val near=RawNearDto(raw);check(codec.decodeFromString<RawNearDto>(codec.encodeToString(near))==near)
        }
        val wideRaw=JsonObject((0 until 240).associate{"p$it" to JsonNull})
        val wide=codec.decodeFromJsonElement(RawWideDto.serializer(),wideRaw)
        check(wide.p0===JsonNull && wide.p239===JsonNull)
        check(codec.encodeToJsonElement(wide)==wideRaw)
        var calls=0;var expected: JsonElement=JsonNull;var bad=false
        val channels=CopyOnWriteArrayList<ByteChannel>()
        val producers=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=StreamingTestEngine(MockEngineConfig().apply {
            addHandler {request ->
                calls++
                when(request.url.encodedPath){
                    "/alias","/direct","/native","/other"->{check(codec.parseToJsonElement(rawAliasBody(request.body))==expected);respond(if(bad) "{" else expected.toString(),HttpStatusCode.OK,headersOf("Content-Type","application/json"))}
                    "/patch"->{check(codec.parseToJsonElement(rawAliasBody(request.body))==expected);respond("",HttpStatusCode.NoContent)}
                    "/events"->{val channel=ByteChannel(autoFlush=true);channels+=channel;val broken=bad
                        producers.launch {try {
                            val frame=if(broken) "event: raw\r\ndata: {\r\n\r\n" else values.joinToString(""){"event: raw\r\nid: raw-id\r\ndata: $it\r\n\r\n"}
                            for(byte in frame.encodeToByteArray())channel.writeFully(byteArrayOf(byte))
                            if(broken)awaitCancellation()
                        } finally {channel.close()}}
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else->error("Unexpected raw alias route")
                }
            }
        })
        val http=HttpClient(engine){install(SSE);install(ContentNegotiation){json(codec)}}
        try {
            val api=RootRawAliasApiClient(http,"http://example.test/",codec).default
            for(value in values){expected=value;check(api.echo(value)==value);check(api.echo2(value)==value);check(api.echo3(value)==value);check(api.echo4(value)==value)}
            bad=true;rawAliasReject {api.echo(expected)};bad=false
            for(state in listOf<Patch<JsonElement>>(Patch.Absent,Patch.Null,Patch.Value(values[2]),Patch.Value(JsonNull))){
                val input=SubmitInputDto(raw=state)
                val body=codec.encodeToJsonElement(input) as JsonObject
                check(when(state){Patch.Absent->"raw" !in body;Patch.Null->body["raw"]==JsonNull;is Patch.Value->body["raw"]==state.value})
                val decoded=codec.decodeFromJsonElement(SubmitInputDto.serializer(),body)
                check(decoded.raw==if(state is Patch.Value && state.value===JsonNull)Patch.Null else state)
                expected=body;api.submit(input)
            }
            val before=calls;val cold=api.watch();check(calls==before)
            repeat(2){val events=withTimeout(10_000){cold.toList()};check(events.map{(it as WatchEventDto.Raw).data}==values);check(events.all{it.id=="raw-id"})}
            check(calls==before+2)
            bad=true;rawAliasReject {withTimeout(10_000){api.watch().toList()}};check(channels.last().isClosedForRead)
            bad=false;check(withTimeout(10_000){api.watch().toList().size}==values.size)
            expected=values[2];check(api.echo(expected)==expected)
        } finally {http.close();engine.close();producers.cancel()}
        println("Root raw alias contracts passed: defaults=$defaults/nulls=$nulls; JsonElement allkinds/JsonNull/Patch/erasure/240+48/HTTP/live SSE close-reuse-cold")
    }
}
