package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerializationException
import kotlinx.serialization.builtins.serializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import org.example.presence.PresenceApiClient
import org.example.presence.api.models.DictionaryInputDtoValue
import org.example.presence.api.models.ModifyInputDto
import org.example.presencecollisions.PresenceCollisionApiClient
import org.example.presencecollisions.api.models.PostEditInputDto
import org.example.presencecollisions.api.models.PostEdit2InputDto
import org.example.presencecollisions.models.Patch as CollisionPatch
import org.example.presence.models.Patch
import org.example.presence.models.PatchSerializer
import org.example.presence.models.StateDto

private suspend fun presenceCollisionContracts() {
    var calls = 0
    val engine = MockEngine { request ->
        calls++
        val body = Json.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString()).jsonObject
        if (calls == 1) {
            check(request.url.encodedPath == "/edit" && request.url.parameters["x"] == "query")
            check(body["memo"]?.jsonPrimitive?.content == "one")
        } else {
            check(request.url.encodedPath == "/edit/path" && body["memo"] == JsonNull)
        }
        respond("", HttpStatusCode.NoContent)
    }
    val http = HttpClient(engine) { install(ContentNegotiation) { json() } }
    try {
        val api = PresenceCollisionApiClient(http).default
        api.postEdit("query", PostEditInputDto(memo = CollisionPatch.Value("one")))
        api.postEdit2("path", PostEdit2InputDto(memo = CollisionPatch.Null))
        check(calls == 2)
    } finally {
        http.close()
        engine.close()
    }
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun requestPresenceContracts() {
    presenceCollisionContracts()
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val absent = ModifyInputDto(token = null)
        val clear = ModifyInputDto(
            token = null, title = Patch.Null, state = Patch.Null, inherited = Patch.Null,
            child = ModifyInputDto.Child(memo = Patch.Null),
            rows = listOf(ModifyInputDto.RowsInner(memo = Patch.Value("row"))),
            attributes = mapOf("a" to ModifyInputDto.AttributesValue(flag = Patch.Null)),
        )
        val value = ModifyInputDto(token = "required", title = Patch.Value("set"), state = Patch.Value(StateDto.Serializer.fromSerialName("future")))
        val absentJson = codec.encodeToJsonElement(ModifyInputDto.serializer(), absent).jsonObject
        check("title" !in absentJson && "state" !in absentJson && "inherited" !in absentJson)
        check(("disabled" in absentJson) == (defaults && nulls))
        check(("token" in absentJson) == nulls)
        check(("note" in absentJson) == (defaults && nulls))
        val clearJson = codec.encodeToJsonElement(ModifyInputDto.serializer(), clear).jsonObject
        check(clearJson["title"] == JsonNull && clearJson["state"] == JsonNull && clearJson["inherited"] == JsonNull)
        check(clearJson["child"]?.jsonObject?.get("memo") == JsonNull)
        check(clearJson["rows"]?.jsonArray?.single()?.jsonObject?.get("memo")?.jsonPrimitive?.content == "row")
        check(clearJson["attributes"]?.jsonObject?.get("a")?.jsonObject?.get("flag") == JsonNull)
        val decoded = codec.decodeFromString(ModifyInputDto.serializer(), """{"token":null,"title":null,"state":"future"}""")
        check(decoded.title === Patch.Null && decoded.state.valueOrNull?.serialName == "future")
        check(codec.decodeFromString(ModifyInputDto.serializer(), """{"token":null}""").title === Patch.Absent)
        val missingRequired = runCatching { codec.decodeFromString(ModifyInputDto.serializer(), "{}") }
        check(missingRequired.isFailure == nulls)
        if (nulls) check(missingRequired.exceptionOrNull() is SerializationException)
        val serializer = PatchSerializer(String.serializer())
        check(codec.encodeToString(serializer, Patch.Null) == "null")
        check(codec.encodeToString(serializer, Patch.Value("x")) == "\"x\"")
        check(codec.encodeToString(Patch.serializer(String.serializer()), Patch.Value("x")) == "\"x\"")
        check(codec.decodeFromString(serializer, "null") === Patch.Null)
        check(runCatching { codec.encodeToString(serializer, Patch.Absent) }.exceptionOrNull() is SerializationException)
        check(Patch.ofNullable<String>(null) === Patch.Null && Patch.orAbsent<String>(null) === Patch.Absent)
        check(Patch.ofNullable("x").valueOrNull == "x" && Patch.orAbsent("x") == Patch.Value("x"))
        val bodies = mutableListOf<JsonObject>()
        var dictionaryCalls = 0
        val engine = MockEngine { request ->
            val payload = codec.parseToJsonElement((request.body as OutgoingContent.ByteArrayContent).bytes().decodeToString())
            when (request.url.encodedPath) {
                "/record" -> {
                    check(request.method == HttpMethod.Patch)
                    bodies += payload.jsonObject
                    respond("""{"title":null}""", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                "/dictionary" -> {
                    dictionaryCalls++
                    check(payload.jsonObject["k"]?.jsonObject?.get("memo") == JsonNull)
                    respond("", HttpStatusCode.NoContent)
                }
                "/value-map" -> {
                    dictionaryCalls++
                    check(payload.jsonObject["k"] == JsonNull)
                    respond("", HttpStatusCode.NoContent)
                }
                "/list" -> {
                    dictionaryCalls++
                    check(payload.jsonArray.map { it.jsonPrimitive.content } == listOf("a", "b"))
                    respond("", HttpStatusCode.NoContent)
                }
                else -> error("Unexpected request")
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            org.example.presenceexcluded.PresenceApiClient(http)
            val api = PresenceApiClient(http).default
            check(api.modify(absent).title == null)
            check(api.modify(clear).title == null)
            check(api.modify(value).title == null)
            check(bodies == listOf(absent, clear, value).map { codec.encodeToJsonElement(ModifyInputDto.serializer(), it).jsonObject })
            api.dictionary(mapOf("k" to DictionaryInputDtoValue(memo = Patch.Null)))
            api.valueMap(mapOf("k" to null))
            api.sendList(listOf("a", "b"))
            check(dictionaryCalls == 3)
            println("Request presence contracts passed: defaults=$defaults/nulls=$nulls, absent/null/value, required nullable, nested/list/map, shared response and bare Patch serializer")
        } finally {
            http.close()
            engine.close()
        }
    }
}
