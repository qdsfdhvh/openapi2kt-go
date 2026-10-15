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
import kotlinx.datetime.LocalDate
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.rootspecial.RootSpecialAliasApiClient
import org.example.rootspecial.api.models.SubmitInputDto
import org.example.rootspecial.api.models.WatchEventDto
import org.example.rootspecial.models.AIdDto
import org.example.rootspecial.models.ALongIdDto
import org.example.rootspecial.models.AStatusDto
import org.example.rootspecial.models.HolderDto
import org.example.rootspecial.models.Patch
import org.example.rootspecial.models.ZIdDto
import org.example.rootspecial.models.ZStatusDto
import org.example.rootspecialprobes.models.ANearDto
import org.example.rootspecialprobes.models.ABoxedDto
import org.example.scalarvalues.Money

private fun rootSpecialReject(block: () -> Unit) {
    try { block(); error("Missing special alias check") }
    catch (cause: IllegalArgumentException) { check(cause.message?.contains("rawValue") == true) }
}
private suspend fun rootSpecialWireReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing special alias wire error") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any {
        (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
    }) { "Wrong special alias error $cause" } }
}
private fun rootSpecialBody(body: OutgoingContent): ByteArray = when (body) {
    is OutgoingContent.ContentWrapper -> rootSpecialBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes()
    else -> error("Unexpected special alias request body")
}

suspend fun rootSpecialAliasContracts() {
    val id = AIdDto("valid")
    check(id == ZIdDto("valid") && id.rawValue == "valid")
    check(AIdDto.serializer().descriptor == ZIdDto.serializer().descriptor)
    val status: AStatusDto = AStatusDto.fromSerialName("ready")
    check(status == ZStatusDto.READY && AStatusDto.entries == ZStatusDto.entries)
    val unknown: AStatusDto = ZStatusDto.Unrecognized("later") // Nested classifiers keep canonical owner.
    rootSpecialReject { AIdDto("") }
    rootSpecialReject { ALongIdDto(-1) }
    check(runCatching { HolderDto(status, id, ALongIdDto(1), 99, LocalDate.parse("2024-01-02"), Long.MAX_VALUE, byteArrayOf(1,-1), Money("12.50"), listOf(status, unknown), mapOf("id" to id), listOf(3)) }.exceptionOrNull()?.message?.contains("enum") == true)
    val value = HolderDto(status, id, ALongIdDto(1), 1, LocalDate.parse("2024-01-02"), Long.MAX_VALUE, byteArrayOf(1,-1), Money("12.50"), listOf(status, unknown), mapOf("id" to id), listOf(3))
    check(value.pin == 7)
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults=defaults; explicitNulls=nulls; ignoreUnknownKeys=true }
        val streamCodec = Json(codec) { ignoreUnknownKeys=false }
        check(codec.decodeFromString<AStatusDto>(codec.encodeToString<AStatusDto>(unknown)) == unknown)
        val text = codec.encodeToString(value)
        val encoded = codec.parseToJsonElement(text) as JsonObject
        check(encoded["big"].toString() == "\"${Long.MAX_VALUE}\"")
        check(encoded["bigList"].toString() == "[3]" && encoded["bytes"].toString() == "[1,-1]" && encoded["pin"].toString() == "7")
        check(runCatching { codec.decodeFromString<HolderDto>(text.replace("\"number\":1", "\"number\":99")) }.exceptionOrNull()?.message?.contains("enum") == true)
        val back = codec.decodeFromString<HolderDto>(text)
        check(back.copy(bytes=value.bytes) == value && back.bytes.contentEquals(value.bytes))
        rootSpecialReject { codec.decodeFromString<HolderDto>(text.replace("\"valid\"", "\"\"")) }
        check(codec.decodeFromString<ANearDto>("{}").pin == 7)
        check(codec.decodeFromString<ANearDto>("{\"pin\":9}").pin == 9) // Const default is not membership enforcement.
        check(codec.decodeFromString<ABoxedDto>("{\"p0\":0}").p0?.rawValue == 0L)
        rootSpecialReject { codec.decodeFromString<ABoxedDto>("{\"p0\":-1}") }
        var calls = 0
        var mode = 0
        var expected = byteArrayOf()
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob()+Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/big", "/bytes" -> {
                        check(rootSpecialBody(request.body).contentEquals(expected))
                        val bytes = if (mode == 1) "\"\"".encodeToByteArray() else expected
                        respond(bytes,HttpStatusCode.OK,headersOf("Content-Type","application/json"))
                    }
                    "/patch" -> { check(rootSpecialBody(request.body).contentEquals(expected));respond("",HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush=true); channels += channel
                        val current = mode
                        producers.launch {
                            try {
                                val frame = when (current) {
                                    1 -> "event: id\r\ndata: \"\"\r\n\r\n"
                                    2 -> "event: status\r\ndata: {}\r\n\r\n"
                                    3 -> "event: id\r\ndata: malformed\r\n\r\n"
                                    else -> "event: id\r\nid: special-id\r\ndata: \"valid\"\r\n\r\nevent: status\r\ndata: \"later\"\r\n\r\n"
                                }
                                for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                                if (current!=0) awaitCancellation()
                            } finally {channel.close()}
                        }
                        respond(channel,HttpStatusCode.OK,headersOf("Content-Type","text/event-stream"))
                    }
                    else -> error("Unexpected special alias path")
                }
            }
        })
        val http = HttpClient(engine) {install(SSE);install(ContentNegotiation){json(codec)}}
        try {
            val api = RootSpecialAliasApiClient(http,"http://example.test/",streamCodec).default
            expected=codec.encodeToString(id).encodeToByteArray();check(api.echo(id)==id);check(api.echo2(id)==id)
            mode=1;rootSpecialWireReject("minLength"){api.echo(id)};mode=0
            expected="${Long.MAX_VALUE}".encodeToByteArray();check(api.echoBig(Long.MAX_VALUE)==Long.MAX_VALUE)
            expected=byteArrayOf(1,-1);check(api.echoBytes(expected).contentEquals(expected))
            for (state in listOf<Patch<ZIdDto>>(Patch.Absent,Patch.Null,Patch.Value(id))) {
                val big: Patch<Long> = when (state){Patch.Absent->Patch.Absent;Patch.Null->Patch.Null;is Patch.Value->Patch.Value(Long.MAX_VALUE)}
                val input=SubmitInputDto(id=state,big=big)
                val body=codec.encodeToJsonElement(input) as JsonObject
                check(body["pin"].toString()=="7")
                check(when(state){Patch.Absent->"id" !in body && "big" !in body;Patch.Null->body["id"]==JsonNull && body["big"]==JsonNull;is Patch.Value->body["id"].toString()=="\"valid\"" && body["big"].toString()=="\"${Long.MAX_VALUE}\""})
                check(codec.decodeFromJsonElement(SubmitInputDto.serializer(),body)==input)
                expected=codec.encodeToString(input).encodeToByteArray();api.submit(input)
            }
            val before=calls;val cold=api.watch();check(calls==before)
            repeat(2){val events=withTimeout(10_000){cold.toList()};check((events[0] as WatchEventDto.Id).data==id);check((events[1] as WatchEventDto.Status).data==unknown);check(events.all{it.id=="special-id"})}
            check(calls==before+2)
            for ((bad,message) in listOf(1 to "minLength",2 to "enum",3 to "JSON")){mode=bad;rootSpecialWireReject(message){withTimeout(10_000){api.watch().toList()}};check(channels.last().isClosedForRead)}
            mode=0;check(withTimeout(10_000){api.watch().toList().size}==2)
            expected=codec.encodeToString(id).encodeToByteArray();check(api.echo(id)==id)
        } finally {http.close();engine.close();producers.cancel()}
        println("Root special alias contracts passed: defaults=$defaults/nulls=$nulls; canonical enum/wrapper/native/const/slots/source/Patch/HTTP/live SSE close-reuse-cold")
    }
}
