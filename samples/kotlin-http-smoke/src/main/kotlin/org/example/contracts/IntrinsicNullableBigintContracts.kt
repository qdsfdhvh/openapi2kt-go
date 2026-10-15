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
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.builtins.serializer
import org.example.intrinsicnullablebigints.models.NativeMaybeDto
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import org.example.intrinsicnullablebigints.IntrinsicNullableBigintApiClient
import org.example.intrinsicnullablebigints.api.models.SubmitInputDto
import org.example.intrinsicnullablebigints.api.models.WatchEventDto
import org.example.intrinsicnullablebigints.models.AInheritedDto
import org.example.intrinsicnullablebigints.models.AFalseDto
import org.example.intrinsicnullablebigints.models.ATrueDto
import org.example.intrinsicnullablebigints.models.HolderDto
import org.example.intrinsicnullablebigints.models.Patch
import org.example.intrinsicnullablebigints.models.ZAmountDto
import org.example.intrinsicnullablebigints.models.ZOmittedDto
import org.example.intrinsicnullablebigints.models.LInheritedDto
import org.example.intrinsicnullablebigints.models.LFalseDto
import org.example.intrinsicnullablebigintprobes.models.ANearDto as BigintNearDto
import org.example.intrinsicnullablebigintprobes.models.ANarrowDto as BigintNearNarrowDto
import org.example.intrinsicnullablebigintprobes.models.AWideDto as BigintWideDto
import org.example.intrinsicnullablebigintprobes.models.ARequiredDto as BigintRequiredDto

private fun intrinsicBigintBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> intrinsicBigintBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected nullable bigint body $body")
}
private suspend fun intrinsicBigintReject(block: suspend () -> Unit) {
    try { block(); error("Invalid bigint kind accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is SerializationException }) { "Wrong nullable bigint failure $cause" } }
}
private suspend fun intrinsicBigintBoundsReject(block: suspend () -> Unit) {
    try { block(); error("Invalid wrapped value accepted") }
    catch (cause: Throwable) { check(generateSequence(cause) { it.cause }.any { it is IllegalArgumentException || it is SerializationException }) { "Wrong bounds failure $cause" } }
}
suspend fun intrinsicNullableBigintContracts() {
    check(serializer<ZAmountDto>().descriptor.isNullable && serializer<ZOmittedDto>().descriptor.isNullable)
    check(!ZAmountDto.serializer().descriptor.isNullable && !ZOmittedDto.serializer().descriptor.isNullable)
    check(AInheritedDto.MIN_VALUE == Long.MIN_VALUE)
    check(NativeMaybeDto.MIN_VALUE == AFalseDto.MIN_VALUE)
    check(!AInheritedDto.serializer().descriptor.isNullable && serializer<AInheritedDto>().descriptor.isNullable)
    check(!serializer<AFalseDto>().descriptor.isNullable && serializer<ATrueDto>().descriptor.isNullable)
    check(!LInheritedDto.serializer().descriptor.isNullable && serializer<LInheritedDto>().descriptor.isNullable)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val ready = Long.MIN_VALUE
        val future = Long.MAX_VALUE
        val amount = 42L
        val nullable = serializer<AInheritedDto>()
        for (value in listOf(null, amount)) check(codec.decodeFromString<LInheritedDto>(codec.encodeToString<LInheritedDto>(value)) == value)
        check(AInheritedDto.MAX_VALUE == Long.MAX_VALUE && LInheritedDto.MIN_VALUE == Long.MIN_VALUE)
        intrinsicBigintReject { codec.decodeFromString<LInheritedDto>("9223372036854775808") }
        for (value in listOf(null, ready, future)) {
            check(codec.encodeToString(nullable, value) == (value?.toString() ?: "null"))
            check(codec.decodeFromString(nullable, codec.encodeToString(nullable, value)) == value)
            check(codec.decodeFromString(ZAmountDto.serializer().nullable, codec.encodeToString(nullable, value)) == value)
        }
        intrinsicBigintReject { codec.decodeFromString(nullable, "{}") }
        val holder = HolderDto(null, ready, future, null, ready, listOf(null, ready, future), mapOf("nil" to null, "future" to future), null, amount, amount, ready, null, ready, null, ready, null, amount)
        val holderJson = codec.encodeToJsonElement(holder) as JsonObject
        check(holderJson["forcedFalse"] == JsonPrimitive(ready.toString()))
        check(holderJson["forcedTrue"] == JsonPrimitive(future.toString()))
        check((holderJson["items"] as JsonArray)[1] == JsonPrimitive(ready))
        check((holderJson["values"] as JsonObject)["future"] == JsonPrimitive(future))
        check(codec.decodeFromString<HolderDto>(codec.encodeToString(holder)) == holder)
        check(codec.decodeFromString<BigintNearDto>("{\"v\":null}").v == null)
        check(codec.decodeFromString<BigintNearNarrowDto>("{\"v\":\"9223372036854775807\"}").v == future)
        intrinsicBigintReject { codec.decodeFromString<BigintNearNarrowDto>("{\"v\":null}") }
        val wide = codec.decodeFromJsonElement(BigintWideDto.serializer(), JsonObject((0 until 240).associate { "p$it" to if (it == 239) JsonPrimitive(ready.toString()) else JsonNull }))
        check(wide.p0 == null && wide.p239 == ready)
        check(codec.decodeFromString<BigintWideDto>(codec.encodeToString(wide)) == wide)
        val allRequired = codec.decodeFromJsonElement(BigintRequiredDto.serializer(), JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(ready.toString()) }))
        check(allRequired.p0 == ready && allRequired.p119 == ready)
        check(codec.decodeFromString<BigintRequiredDto>(codec.encodeToString(allRequired.copy(p119 = future))) == allRequired.copy(p119 = future))
        val absent = SubmitInputDto(null, ready)
        val encoded = codec.encodeToJsonElement(absent) as JsonObject
        check("inherited" !in encoded && "wrapper" !in encoded && "forcedTrue" !in encoded)
        check(if (nulls) encoded["requiredNull"] === JsonNull else "requiredNull" !in encoded)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(absent)) == absent)
        if (nulls) intrinsicBigintReject { codec.decodeFromString<SubmitInputDto>("{\"requiredFalse\":\"-9223372036854775808\"}") }
        val clear = absent.copy(inherited = Patch.Null, wrapper = Patch.Null, forcedTrue = Patch.Null, amount = Patch.Null, amountTrue = Patch.Null)
        val clearJson = codec.encodeToJsonElement(clear) as JsonObject
        check(clearJson["inherited"] === JsonNull && clearJson["wrapper"] === JsonNull)
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(clear)) == clear)
        val directClear = absent.copy(direct = Patch.Null, directWrapper = Patch.Null, directOmitted = Patch.Null)
        val directValue = absent.copy(direct = Patch.Value(future), directWrapper = Patch.Value(ready), directOmitted = Patch.Value(amount))
        check((codec.encodeToJsonElement(directValue) as JsonObject)["direct"] == JsonPrimitive(future.toString()))
        for (direct in listOf(directClear, directValue)) check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(direct)) == direct)
        val value = absent.copy(inherited = Patch.Value(future), wrapper = Patch.Value(ready), forcedTrue = Patch.Value(future), amount = Patch.Value(amount), amountTrue = Patch.Value(amount))
        check((codec.encodeToJsonElement(value) as JsonObject)["inherited"] == JsonPrimitive(future.toString()))
        check(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(value)) == value)
        intrinsicBigintReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"-9223372036854775808\",\"inherited\":{}}") }
        intrinsicBigintBoundsReject { codec.decodeFromString<SubmitInputDto>("{\"requiredNull\":null,\"requiredFalse\":\"-9223372036854775808\",\"amount\":9223372036854775808}") }
        var expected: JsonElement = JsonNull
        var failure = ""
        var calls = 0
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/alias", "/direct", "/optional", "/amount", "/amount-direct" -> { check(codec.parseToJsonElement(intrinsicBigintBody(request.body)) == expected); respond(if (failure == "kind") "{}" else if (failure == "bounds") "9223372036854775808" else if (failure == "amountBounds") "9223372036854775808" else if (failure == "syntax") "{" else expected.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json")) }
                    "/query" -> { check(request.url.parameters["amount"] == amount.toString()); check(request.url.parameters["since"] == (if (failure == "noSince") null else ready.toString())); respond("", HttpStatusCode.NoContent) }
                    "/patch" -> { check(codec.parseToJsonElement(intrinsicBigintBody(request.body)) == expected); respond("", HttpStatusCode.NoContent) }
                    "/events" -> {
                        val channel = ByteChannel(autoFlush = true); channels += channel; val broken = failure
                        producers.launch { try {
                            val frame = when (broken) {
                                "narrow" -> "event: narrow\r\ndata: null\r\n\r\n"
                                "bounds" -> "event: nullable\r\ndata: 9223372036854775808\r\n\r\n"
                                "amountBounds" -> "event: amount\r\ndata: 9223372036854775808\r\n\r\n"
                                "amountNarrow" -> "event: amountNarrow\r\ndata: null\r\n\r\n"
                                "kind" -> "event: nullable\r\ndata: {}\r\n\r\n"
                                "syntax" -> "event: nullable\r\ndata: {\r\n\r\n"
                                else -> "event: nullable\r\ndata: null\r\n\r\nevent: nullable\r\ndata: 9223372036854775807\r\n\r\nevent: narrow\r\ndata: -9223372036854775808\r\n\r\nevent: amount\r\ndata: null\r\n\r\nevent: amountNarrow\r\ndata: 42\r\n\r\n"
                            }
                            for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            if (broken.isNotEmpty()) awaitCancellation()
                        } finally { channel.close() } }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected nullable bigint route")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = IntrinsicNullableBigintApiClient(http, "http://example.test/", codec).default
            check(api.echo(null) == null && api.optional() == null)
            api.query(amount = amount, since = ready)
            failure = "noSince"; api.query(amount = amount); failure = ""
            for (wrapped in listOf(ready, future)) { expected = codec.encodeToJsonElement(ZAmountDto.serializer(), wrapped); check(api.echo(wrapped) == wrapped && api.echo2(wrapped) == wrapped && api.optional(wrapped) == wrapped) }
            expected = JsonNull; check(api.echoAmount(null) == null)
            expected = codec.encodeToJsonElement(ZOmittedDto.serializer(), amount); check(api.echoAmount(amount) == amount && api.echoAmount2(amount) == amount)
            for (patch in listOf(absent, clear, value, directClear, directValue)) { expected = codec.encodeToJsonElement(patch); api.submit(patch) }
            expected = codec.encodeToJsonElement(ZAmountDto.serializer(), ready)
            for (bad in listOf("kind", "syntax")) { failure = bad; intrinsicBigintReject { api.echo(ready) } }
            failure = "bounds"; intrinsicBigintBoundsReject { api.echo(ready) }
            expected = codec.encodeToJsonElement(ZOmittedDto.serializer(), amount); failure = "amountBounds"; intrinsicBigintBoundsReject { api.echoAmount(amount) }
            expected = codec.encodeToJsonElement(ZAmountDto.serializer(), ready)
            failure = ""; val before = calls; val flow = api.watch(); check(calls == before)
            val events = withTimeout(3000) { flow.toList() }
            check(events.size == 5 && (events[0] as WatchEventDto.Nullable).data == null)
            check((events[1] as WatchEventDto.Nullable).data == future && (events[2] as WatchEventDto.Narrow).data == ready)
            check((events[3] as WatchEventDto.Amount).data == null && (events[4] as WatchEventDto.AmountNarrow).data == amount)
            for (bad in listOf("bounds", "amountBounds")) { failure = bad; withTimeout(3000) { intrinsicBigintBoundsReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            for (bad in listOf("narrow", "amountNarrow", "kind", "syntax")) { failure = bad; withTimeout(3000) { intrinsicBigintReject { api.watch().toList() } }; withTimeout(3000) { while (!channels.last().isClosedForWrite) kotlinx.coroutines.yield() } }
            failure = ""; check(api.echo2(ready) == ready)
        } finally { http.close(); producers.cancel() }
        println("IntrinsicNullableBigint Json defaults=$defaults nulls=$nulls checks passed")
    }
}
