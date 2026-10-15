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
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.take
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.coroutines.yield
import kotlinx.serialization.SerializationException
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.borrowedproductunionleaves.BorrowedCollectionProductUnionLeavesApiClient
import org.example.borrowedproductunionleaves.api.models.SubmitInputDto
import org.example.borrowedproductunionleaves.api.models.WatchEventDto
import org.example.borrowedproductunionleaves.models.AgainBridgeDto
import org.example.borrowedproductunionleaves.models.CollectionChoiceDto
import org.example.borrowedproductunionleaves.models.HolderDto
import org.example.borrowedproductunionleaves.models.RowsAliasDto
import org.example.borrowedproductunionleaves.models.RowsBridgeDto
import org.example.borrowedproductunionleaves.models.MapAliasDto
import org.example.borrowedproductunionleaves.models.MapBridgeDto
import org.example.borrowedproductunionleaves.models.NodeDto
import org.example.borrowedproductunionleaves.models.OwnerDto
import org.example.borrowedproductunionleaves.models.UnknownNodeDto
import org.example.borrowedproductunionleaves.models.NativeRootDto
import org.example.borrowedproductunionleaves.models.StringNativeRootDto
import org.example.borrowedproductunionleaves.models.BooleanNativeRootDto
import org.example.borrowedproductunionleaves.models.NullNativeRootDto
import org.example.borrowedproductunionleaves.models.NativeRowsBridgeDto
import org.example.borrowedproductunionleaves.models.NativeNullableRowsBridgeDto
import org.example.borrowedproductunionleaves.models.Patch

@OptIn(io.ktor.utils.io.InternalAPI::class)
private class BorrowedProductUnionStreamingEngine(config: MockEngineConfig) : MockEngine(config) {
    override val supportedCapabilities = super.supportedCapabilities + SSECapability
    override suspend fun execute(data: HttpRequestData): HttpResponseData {
        val response = super.execute(data)
        val adapter = data.attributes.getOrNull(ResponseAdapterAttributeKey) ?: return response
        val body = adapter.adapt(data, response.statusCode, response.headers, response.body as ByteReadChannel, data.body, response.callContext) ?: return response
        return HttpResponseData(response.statusCode, response.requestTime, response.headers, response.version, body, response.callContext)
    }
}
private fun borrowedProductUnionBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> borrowedProductUnionBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected synthetic borrowed collection body ${body::class}")
}
private suspend fun borrowedProductUnionReject(message: String, block: suspend () -> Unit) {
    try { block(); error("Missing borrowed collection rejection: $message") }
    catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) && it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong borrowed collection failure: $cause" }
    }
}

suspend fun borrowedCollectionProductUnionLeavesContracts(checked: Boolean = false) = withTimeout(30000) {
    check(serializer<RowsAliasDto>().descriptor.isNullable && !serializer<AgainBridgeDto>().descriptor.isNullable)
    check(!serializer<RowsBridgeDto>().descriptor.isNullable)
    check(serializer<MapAliasDto>().descriptor.isNullable && !serializer<MapBridgeDto>().descriptor.isNullable)
    val values: List<NodeDto> = listOf(NodeDto.Id("parent", listOf(NodeDto.Id("child"))), NodeDto.Name("named"))
    val mapping: MapBridgeDto = mapOf("key" to OwnerDto.StringNode("nested"), "nil" to null)
    fun holder(rows: List<NodeDto> = values, map: MapBridgeDto = mapping) = HolderDto(rows = rows, mapping = map, direct = values)
    val model = holder()
    if (checked) {
        borrowedProductUnionReject("minItems") { holder(emptyList()) }
        borrowedProductUnionReject("maxItems") { holder(List(4) { NodeDto.Id("ok") }) }
        borrowedProductUnionReject("minLength") { NodeDto.Id("a") }
        borrowedProductUnionReject("minProperties") { holder(map = emptyMap()) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val unknownWire = """{"kind":"future","extra":[true,null]}"""
        val unknown: NodeDto = codec.decodeFromString<NodeDto>(unknownWire)
        check(unknown is UnknownNodeDto)
        check(codec.parseToJsonElement(codec.encodeToString<NodeDto>(unknown)) == codec.parseToJsonElement(unknownWire))
        val native: NativeRowsBridgeDto = listOf(StringNativeRootDto("wire"), BooleanNativeRootDto(true), NullNativeRootDto)
        check(codec.decodeFromString<NativeRowsBridgeDto>(codec.encodeToString(native)) == native)
        check(codec.decodeFromString(NativeRootDto.serializer(), "null") === NullNativeRootDto)
        val nativeNullable: NativeNullableRowsBridgeDto = native
        check(codec.decodeFromString<NativeNullableRowsBridgeDto>(codec.encodeToString(nativeNullable)) == listOf(native[0], native[1], null))
        borrowedProductUnionReject("Unexpected JSON kind") { codec.decodeFromString(OwnerDto.Node.serializer(), "null") }
        check(codec.decodeFromString<OwnerDto.Node?>("null") == null)
        val nestedProduct: org.example.borrowedproductunionleaves.models.ProductOwnerDto.Choice = org.example.borrowedproductunionleaves.models.ProductOwnerDto.Choice.Id("parent", listOf(NodeDto.Id("child")))
        val nestedViews = org.example.borrowedproductunionleaves.models.ProductViewsDto(mapOf("value" to nestedProduct, "nil" to null))
        check(codec.encodeToJsonElement(codec.decodeFromString<org.example.borrowedproductunionleaves.models.ProductViewsDto>(codec.encodeToString(nestedViews))) == codec.encodeToJsonElement(nestedViews))
        check(codec.decodeFromString(org.example.borrowedproductunionleaves.models.ProductOwnerDto.Choice.serializer(), codec.encodeToString(org.example.borrowedproductunionleaves.models.ProductOwnerDto.Choice.serializer(), nestedProduct)) is org.example.borrowedproductunionleaves.models.ProductOwnerDto.Choice.Id)
        val alias: RowsAliasDto = values
        val again: AgainBridgeDto = values
        check(codec.decodeFromString<RowsAliasDto>(codec.encodeToString(alias)) == values)
        check(codec.decodeFromString<AgainBridgeDto>(codec.encodeToString(again)) == values)
        check(codec.decodeFromString<RowsAliasDto>("null") == null)
        check(codec.decodeFromString<MapBridgeDto>(codec.encodeToString(mapping)) == mapping)
        check(codec.decodeFromString<MapAliasDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(model))) == codec.encodeToJsonElement(model))
        borrowedProductUnionReject("null") { codec.decodeFromString<RowsBridgeDto>("null") }
        borrowedProductUnionReject("JSON") { codec.decodeFromString<RowsAliasDto>("{}") }
        val unsafe: RowsBridgeDto = codec.decodeFromString(codec.encodeToString(values))
        if (checked) borrowedProductUnionReject("minLength") { codec.decodeFromString<RowsBridgeDto>(codec.encodeToString(values).replace("parent", "a")) }
        check(((unsafe.first() as NodeDto.Id).children?.first() as NodeDto.Id).id == "child")
        check((codec.decodeFromString(NodeDto.serializer(), codec.encodeToString(NodeDto.serializer(), values.first())) as NodeDto.Id).id == "parent")
        check((codec.decodeFromString(OwnerDto.Node.serializer(), codec.encodeToString(OwnerDto.Node.serializer(), mapping.getValue("key")!!)) as OwnerDto.StringNode).value == "nested")
        val listChoice: CollectionChoiceDto = CollectionChoiceDto.Rows(values)
        val mapChoice: CollectionChoiceDto = CollectionChoiceDto.Map(mapping)
        for (choice in listOf(listChoice, mapChoice)) check(codec.encodeToJsonElement(codec.decodeFromString<CollectionChoiceDto>(codec.encodeToString(choice))) == codec.encodeToJsonElement(choice))
        if (checked) borrowedProductUnionReject("minItems") { CollectionChoiceDto.Rows(emptyList()) }
        val absent = SubmitInputDto(requiredNull = null, required = values)
        val clear = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Null, map = Patch.Null)
        val assigned = SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(values), map = Patch.Value(mapping))
        for (input in listOf(absent, clear, assigned)) check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) == codec.encodeToJsonElement(input))
        check("optional" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["optional"] == JsonNull)
        if (nulls) borrowedProductUnionReject("requiredNull") { codec.decodeFromJsonElement(SubmitInputDto.serializer(), JsonObject((codec.encodeToJsonElement(absent) as JsonObject).filterKeys { it != "requiredNull" })) }
        if (checked) {
            borrowedProductUnionReject("minItems") { SubmitInputDto(requiredNull = null, required = values, optional = Patch.Value(emptyList())) }
            borrowedProductUnionReject("minProperties") { SubmitInputDto(requiredNull = null, required = values, map = Patch.Value(emptyMap())) }
        }
        var expected = "null"
        var mode = 0
        var calls = 0
        var endless = false
        val channels = CopyOnWriteArrayList<ByteChannel>()
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        val engine = BorrowedProductUnionStreamingEngine(MockEngineConfig().apply { addHandler { request ->
            calls++
            when (request.url.encodedPath) {
                "/events" -> {
                    val channel = ByteChannel(autoFlush = true); channels += channel
                    val bad = mode; val keepOpen = endless
                    producers.launch { try {
                        val frame = when (bad) {
                            1 -> "event: map\r\ndata: null\r\n\r\n"
                            2 -> "event: rows\r\ndata: []\r\n\r\n"
                            3 -> "event: rows\r\ndata: [{\"kind\":\"id\",\"stamp\":\"shared\",\"id\":\"a\"}]\r\n\r\n"
                            4 -> "event: rows\r\ndata: {\r\n\r\n"
                            5 -> "event: map\r\ndata: {\"key\":1}\r\n\r\n"
                            else -> "event: rows\r\ndata: ${codec.encodeToString(values)}\r\n\r\nevent: map\r\ndata: ${codec.encodeToString(mapping)}\r\n\r\n"
                        }
                        for (byte in frame.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                        if (bad != 0 || keepOpen) awaitCancellation()
                    } catch (closed: ClosedByteChannelException) { check(channel.isClosedForWrite) }
                    finally { channel.close() } }
                    respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                }
                "/patch" -> { check(codec.parseToJsonElement(borrowedProductUnionBody(request.body)) == codec.parseToJsonElement(expected)); respond("", HttpStatusCode.NoContent) }
                else -> {
                    check(codec.parseToJsonElement(borrowedProductUnionBody(request.body)) == codec.parseToJsonElement(expected))
                    val response = when (mode) { 1 -> "null"; 2 -> if (request.url.encodedPath == "/map") "{}" else "[]"; 3 -> expected.replace("parent", "a").replace("nested", "a"); else -> expected }
                    respond(if (mode == 4) "{\"key\":1}" else response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
            }
        } })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = BorrowedCollectionProductUnionLeavesApiClient(http, "https://example.test/", codec).default
            if (checked) {
                val before = calls; var callback = false
                borrowedProductUnionReject("minItems") { api.echo(emptyList()) { callback = true } }
                borrowedProductUnionReject("minProperties") { api.echoMap(emptyMap()) { callback = true } }
                check(calls == before && !callback)
            }
            expected = codec.encodeToString(values); check(api.echo(values) == values)
            if (checked) { mode = 2; borrowedProductUnionReject("minItems") { api.echo(values) }; mode = 3; borrowedProductUnionReject("minLength") { api.echo(values) } }
            mode = 0; expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
            mode = 1; borrowedProductUnionReject("null") { api.echoMap(mapping) }
            if (checked) { mode = 2; borrowedProductUnionReject("minProperties") { api.echoMap(mapping) } }
            mode = 4; borrowedProductUnionReject("Unexpected JSON kind") { api.echoMap(mapping) }
            mode = 0
            for (input in listOf(absent, clear, assigned)) { expected = codec.encodeToString(input); api.submit(input) }
            val before = calls; val cold = api.watch(); check(calls == before)
            repeat(2) {
                val events = withTimeout(3000) { cold.toList() }
                check((events[0] as WatchEventDto.Rows).data == values && (events[1] as WatchEventDto.Map).data == mapping)
            }
            for ((bad, message) in listOf(1 to "null", 4 to "JSON", 5 to "Unexpected JSON kind") + if (checked) listOf(2 to "minItems", 3 to "minLength") else emptyList()) {
                mode = bad; withTimeout(3000) { borrowedProductUnionReject(message) { cold.toList() } }
                withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            }
            mode = 0; endless = true; withTimeout(3000) { cold.take(1).toList() }
            withTimeout(3000) { while (!channels.last().isClosedForWrite) yield() }
            endless = false; check(withTimeout(3000) { cold.toList() }.size == 2)
            expected = codec.encodeToString(mapping); check(api.echoMap(mapping) == mapping)
        } finally { http.close(); producers.cancel(); channels.forEach { it.cancel(null) } }
        println("BorrowedCollectionProductUnionLeaves Json defaults=$defaults nulls=$nulls actual-union-dispatch-carriers-source-nullable-Patch-union-HTTP-SSEclose-cancel-reuse-cold passed")
    }
}
