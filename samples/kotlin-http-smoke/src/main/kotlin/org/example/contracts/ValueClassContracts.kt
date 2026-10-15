package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngineConfig
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.plugins.sse.SSE
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import io.ktor.utils.io.ByteChannel
import io.ktor.utils.io.writeFully
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.put
import org.example.valueclasses.ValueClassesApiClient
import org.example.valueclasses.api.models.PatchInputDto
import org.example.valueclasses.api.models.RoundTripFilterDto
import org.example.valueclasses.api.models.WatchEventDto
import org.example.valueclasses.models.CounterDto
import org.example.valueclasses.models.EnabledDto
import org.example.valueclasses.models.IndexDto
import org.example.valueclasses.models.OtherIdDto
import org.example.valueclasses.models.Patch
import org.example.valueclasses.models.RatioDto
import org.example.valueclasses.models.RecordDto
import org.example.valueclasses.models.UserIdDto
import org.example.valueclasses.models.WeightDto

// Different wrappers of String must retain distinct overload/runtime identities.
private fun nominal(value: UserIdDto) = "user:${value.rawValue}"
private fun nominal(value: OtherIdDto) = "other:${value.rawValue}"

private fun valueClassBody(content: OutgoingContent): ByteArray = when (content) {
    is OutgoingContent.ContentWrapper -> valueClassBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes()
    else -> error("Unexpected value class body $content")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun valueClassContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(nominal(UserIdDto("same")) == "user:same")
        check(nominal(OtherIdDto("same")) == "other:same")
        check(!UserIdDto("same").equals(OtherIdDto("same")))
        val value = RecordDto(
            id = UserIdDto("a/b"), counter = CounterDto(Long.MIN_VALUE), index = IndexDto(Int.MIN_VALUE),
            enabled = EnabledDto(false), ratio = RatioDto(-0.0), weight = WeightDto(2.5f),
            rows = listOf(UserIdDto("x"), null), bag = mapOf("k" to UserIdDto("v")))
        val tree = codec.encodeToJsonElement(RecordDto.serializer(), value).jsonObject
        check(tree["id"] == JsonPrimitive("a/b"))
        check(tree["counter"] == JsonPrimitive(Long.MIN_VALUE))
        check(tree["index"] == JsonPrimitive(Int.MIN_VALUE))
        check(tree["enabled"] == JsonPrimitive(false))
        check(tree["ratio"] == JsonPrimitive(-0.0))
        check(tree["weight"] == JsonPrimitive(2.5f))
        check(tree["rows"] == JsonArray(listOf(JsonPrimitive("x"), JsonNull)))
        check(tree["bag"] == buildJsonObject { put("k", "v") })
        check(codec.decodeFromJsonElement(RecordDto.serializer(), tree) == value)
        check(codec.encodeToString(CounterDto.serializer(), CounterDto(Long.MAX_VALUE)) == Long.MAX_VALUE.toString())
        check(codec.decodeFromString(CounterDto.serializer(), Long.MIN_VALUE.toString()).rawValue == Long.MIN_VALUE)
        check(codec.decodeFromString(UserIdDto.serializer(), "\"🍃\"").rawValue == "🍃")
        for (bad in listOf("{}", "[]", "null", "{\"rawValue\":\"x\"}")) {
            try { codec.decodeFromString(UserIdDto.serializer(), bad); error("Expected invalid scalar wrapper shape") }
            catch (cause: SerializationException) { /* underlying scalar decoder rejects objects/null */ }
        }
        for ((input, expected) in listOf(
            PatchInputDto() to buildJsonObject {},
            PatchInputDto(Patch.Null) to buildJsonObject { put("id", JsonNull) },
            PatchInputDto(Patch.Value(UserIdDto("patch"))) to buildJsonObject { put("id", "patch") })) {
            val encoded = codec.encodeToJsonElement(PatchInputDto.serializer(), input)
            check(encoded == expected)
            check(codec.decodeFromJsonElement(PatchInputDto.serializer(), encoded) == input)
        }
        val producers = CoroutineScope(SupervisorJob() + Dispatchers.Default)
        var calls = 0
        val engine = StreamingTestEngine(MockEngineConfig().apply {
            addHandler { request ->
                calls++
                when (request.url.encodedPath) {
                    "/items/a%2Fb" -> {
                        if (calls == 1) {
                            check(request.url.parameters["other"] == "opaque-id")
                            check(request.url.parameters.getAll("many") == listOf("x"))
                            check(request.url.parameters["csv"] == "1,9223372036854775807")
                            check(request.url.parameters["kept"] == "2" && "dropped" !in request.url.parameters)
                            check(request.url.parameters["filter[id]"] == "filter")
                            check(request.url.parameters["filter[active]"] == "false")
                            check(request.headers["X-Enabled"] == "false")
                            check(request.headers["X-Ids"] == "x,null")
                        } else {
                            check(request.url.parameters.names().isEmpty())
                            check(request.headers["X-Enabled"] == null && request.headers["X-Ids"] == null)
                        }
                        check(codec.parseToJsonElement(valueClassBody(request.body).decodeToString()) == tree)
                        respond(codec.encodeToString(RecordDto.serializer(), value), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/root" -> {
                        val body = valueClassBody(request.body).decodeToString()
                        check(body == "\"root\"") // wrapper uses JSON serialization, unlike bare String bodies
                        respond(body, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                    }
                    "/patch" -> {
                        check(codec.parseToJsonElement(valueClassBody(request.body).decodeToString()) == buildJsonObject { put("id", "patch") })
                        respond("", HttpStatusCode.NoContent)
                    }
                    "/vc-events" -> {
                        val channel = ByteChannel(autoFlush = true)
                        producers.launch {
                            try {
                                val frames = "event: identifier\ndata: \"🍃\"\n\nevent: count\ndata: 9223372036854775807\n\n"
                                for (byte in frames.encodeToByteArray()) channel.writeFully(byteArrayOf(byte))
                            } finally { channel.close() }
                        }
                        respond(channel, HttpStatusCode.OK, headersOf("Content-Type", "text/event-stream"))
                    }
                    else -> error("Unexpected value class request ${request.url}")
                }
            }
        })
        val http = HttpClient(engine) { install(SSE); install(ContentNegotiation) { json(codec) } }
        try {
            val api = ValueClassesApiClient(http, "http://example.test/", codec).types
            check(api.roundTrip(UserIdDto("a/b"), OtherIdDto("opaque-id"), listOf(UserIdDto("x"), null),
                listOf(CounterDto(1), CounterDto(Long.MAX_VALUE)), mapOf("kept" to CounterDto(2), "dropped" to null),
                RoundTripFilterDto(active = EnabledDto(false), id = UserIdDto("filter")), EnabledDto(false), listOf(UserIdDto("x"), null), value) == value)
            check(api.roundTrip(UserIdDto("a/b"), input = value) == value)
            check(api.root(UserIdDto("root")) == UserIdDto("root"))
            api.patchPatch(PatchInputDto(Patch.Value(UserIdDto("patch"))))
            val events = api.watch().toList()
            check((events[0] as WatchEventDto.Identifier).data == UserIdDto("🍃"))
            check((events[1] as WatchEventDto.Count).data == CounterDto(Long.MAX_VALUE))
            check(calls == 5)
            println("Value class contracts passed: defaults=$defaults/nulls=$nulls; nominal overloads/native JSON/ref/list/map/Patch/rawValue parameters/quoted root wrapper/SSE payload; $calls requests")
        } finally { http.close(); engine.close(); producers.cancel() }
    }
}
