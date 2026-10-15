@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
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
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.ByteReadChannel
import io.ktor.utils.io.ClosedByteChannelException
import io.ktor.utils.io.writeFully
import java.util.concurrent.CopyOnWriteArrayList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.serializer
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import org.example.bigintchains.BigintReferenceChainsApiClient
import org.example.bigintchains.api.models.SubmitInputDto
import org.example.bigintchains.api.models.WatchEventDto
import org.example.bigintchains.models.AmountBridgeDto
import org.example.bigintchains.models.AmountFalseBridgeDto
import org.example.bigintchains.models.AgainBridgeDto
import org.example.bigintchains.models.HolderDto
import org.example.bigintchains.models.BigintChoiceDto
import org.example.bigintchains.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BigintChainStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}

private fun bigintChainBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> bigintChainBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic builtin body")
}

suspend fun bigintReferenceChainsContracts() = withTimeout(20000) {
    check(serializer<AmountBridgeDto>().descriptor.isNullable)
    check(serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<AmountFalseBridgeDto>().descriptor.isNullable)
    check(!AmountBridgeDto.serializer().descriptor.isNullable)
    check(AmountFalseBridgeDto.MAX_VALUE == Long.MAX_VALUE)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val lo=Long.MIN_VALUE;val hi=Long.MAX_VALUE
        for(value in listOf(null,lo,hi)) {
            val wire=codec.encodeToString<AmountBridgeDto>(value)
            check(wire==(value?.toString()?:"null"))
            check(codec.decodeFromString<AmountBridgeDto>(wire)==value)
            check(codec.decodeFromString(Long.serializer().nullable,wire)==value)
        }
        for(bad in listOf("[]","{}","9223372036854775808","{")) check(runCatching {codec.decodeFromString<AmountBridgeDto>(bad)}.isFailure)
        check(runCatching {codec.decodeFromString<AmountFalseBridgeDto>("null")}.isFailure)
        val holder=HolderDto(null,lo,hi,null,lo,listOf(null,lo,hi),mapOf("nil" to null,"future" to hi),null,hi,hi,lo,null,lo,null,lo,null,hi)
        val tree=codec.encodeToJsonElement(holder).jsonObject
        check(tree["forcedFalse"]==JsonPrimitive(lo.toString()))
        check(tree["directFalse"]==JsonPrimitive(lo.toString()))
        check((tree["items"] as JsonArray)[1]==JsonPrimitive(lo))
        check((tree["values"] as JsonObject)["future"]==JsonPrimitive(hi))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))==holder)
        val choice:BigintChoiceDto=BigintChoiceDto.A(hi)
        check(codec.encodeToJsonElement<BigintChoiceDto>(choice).jsonObject["value"]==JsonPrimitive(hi.toString()))
        check(codec.decodeFromString<BigintChoiceDto>(codec.encodeToString<BigintChoiceDto>(choice))==choice)
        val patches=listOf(SubmitInputDto(requiredNull=null,requiredFalse=lo),SubmitInputDto(requiredNull=null,requiredFalse=lo,inherited=Patch.Null,wrapper=Patch.Null),SubmitInputDto(requiredNull=null,requiredFalse=lo,inherited=Patch.Value(hi),wrapper=Patch.Value(lo),amount=Patch.Value(hi)))
        check("inherited" !in codec.encodeToJsonElement(patches[0]).jsonObject)
        check(codec.encodeToJsonElement(patches[1]).jsonObject["inherited"]==JsonNull)
        check(codec.encodeToJsonElement(patches[2]).jsonObject["inherited"]==JsonPrimitive(hi.toString()))
        for(patch in patches) check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(patch))==patch)
        check(runCatching {codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"-9223372036854775808\",\"inherited\":{}}")}.isFailure)
        var failure="";var expected:kotlinx.serialization.json.JsonElement=JsonNull
        var expectedPatch=codec.encodeToJsonElement(patches[0]);var since:Long?=lo;var calls=0
        val channels=CopyOnWriteArrayList<ByteChannel>();val writers=CopyOnWriteArrayList<Job>()
        val scope=CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine=BigintChainStreamingEngine(MockEngineConfig().apply {
            addHandler {request ->
                calls++
                when(request.url.encodedPath) {
                    "/alias","/direct","/optional","/amount","/amount-direct" -> {
                        check(codec.parseToJsonElement(bigintChainBody(request.body))==expected)
                        respond(when(failure) {"overflow"->"9223372036854775808";"null"->"null";"kind"->"[]";"syntax"->"{";else->expected.toString()},HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/query" -> {check(request.url.parameters["amount"]=="42");check(request.url.parameters["since"]==since?.toString());respond("",HttpStatusCode.NoContent)}
                    "/patch" -> {check(codec.parseToJsonElement(bigintChainBody(request.body))==expectedPatch);respond("",HttpStatusCode.NoContent)}
                    "/events" -> {
                        val channel=ByteChannel(autoFlush=true);channels.add(channel)
                        val frame=when(failure) {
                            "null"->"event: narrow\r\ndata: null\r\n\r\n"
                            "overflow"->"event: nullable\r\ndata: 9223372036854775808\r\n\r\n"
                            "kind"->"event: nullable\r\ndata: {}\r\n\r\n"
                            "syntax"->"event: nullable\r\ndata: {\r\n\r\n"
                            else->"event: nullable\r\ndata: null\r\n\r\nevent: narrow\r\ndata: -9223372036854775808\r\n\r\nevent: amount\r\ndata: 9223372036854775807\r\n\r\nevent: amountNarrow\r\ndata: 42\r\n\r\n"
                        }
                        writers.add(scope.launch {try {for(byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte));awaitCancellation()} catch(closed:ClosedByteChannelException) {if(!channel.isClosedForWrite)throw closed}})
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else->error("Unexpected synthetic bigint route")
                }
            }
        })
        val http=HttpClient(engine) {install(ContentNegotiation) {json(codec)};install(SSE)}
        try {
            val api=BigintReferenceChainsApiClient(http,"https://example.test/",codec).default
            check(api.echo(null)==null && api.optional()==null)
            for(value in listOf(lo,hi)) {expected=JsonPrimitive(value);check(api.echo(value)==value && api.echo2(value)==value && api.optional(value)==value)}
            expected=JsonNull;check(api.echoAmount(null)==null)
            expected=JsonPrimitive(42L);check(api.echoAmount(42L)==42L && api.echoAmount2(42L)==42L)
            api.query(amount=42L,since=lo);since=null;api.query(amount=42L)
            for(patch in patches) {expectedPatch=codec.encodeToJsonElement(patch);api.submit(patch)}
            expected=JsonPrimitive(lo)
            for(bad in listOf("overflow","kind","syntax")) {failure=bad;check(runCatching {api.echo(lo)}.isFailure)}
            failure="null";check(runCatching {api.echo2(lo)}.isFailure);failure=""
            val cold=api.watch();check(channels.isEmpty())
            repeat(2) {
                val events=cold.take(4).toList()
                check((events[0] as WatchEventDto.Nullable).data==null)
                check((events[1] as WatchEventDto.Narrow).data==lo)
                check((events[2] as WatchEventDto.Amount).data==hi)
                check((events[3] as WatchEventDto.AmountNarrow).data==42L)
                check(channels.last().isClosedForWrite)
            }
            for(bad in listOf("null","overflow","kind","syntax")) {failure=bad;check(runCatching {cold.take(1).toList()}.isFailure);check(channels.last().isClosedForWrite)}
            failure="";check((cold.take(1).toList().single() as WatchEventDto.Nullable).data==null);check(channels.last().isClosedForWrite)
            check(api.echo2(lo)==lo)
        } finally {http.close();scope.cancel();writers.forEach {it.join()}}
        println("BigintReferenceChains Json defaults=$defaults nulls=$nulls field-string-root-number-Patch-query-HTTP-liveSSEclose-reuse-cold passed")
    }
}
