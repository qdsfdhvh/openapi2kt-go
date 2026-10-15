package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.datetime.LocalDate
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import org.example.specialscalars.ApiClient
import org.example.specialscalars.api.models.SpecialInputDto
import org.example.specialscalars.models.Patch
import org.example.specialscalars.models.RecordDto
import kotlin.time.Instant
import org.example.scalarvalues.Money

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun specialScalarContracts() {
    val stamp = Instant.parse("2024-02-29T12:34:56Z")
    val day = LocalDate.parse("2024-02-29")
    val bytes = byteArrayOf(-128, -1, 0, 127)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val input = SpecialInputDto(stamp, day, bytes, Long.MAX_VALUE, JsonNull,
            maybeBig = Patch.Value(Long.MIN_VALUE), stamps = listOf(stamp), bigList = listOf(Long.MAX_VALUE),
            bigMap = mapOf("large" to Long.MIN_VALUE), days = mapOf("day" to day, "missing" to null),
            child = SpecialInputDto.Child(Patch.Value(7)), money = Money("12.50"))
        val tree = codec.encodeToJsonElement(SpecialInputDto.serializer(), input).jsonObject
        check(tree["stamp"] == JsonPrimitive(stamp.toString()) && tree["day"] == JsonPrimitive(day.toString()))
        check(tree["bytes"] == JsonArray(bytes.map { JsonPrimitive(it) }))
        check(tree["big"] == JsonPrimitive(Long.MAX_VALUE.toString()))
        check(tree["money"] == JsonPrimitive("12.50"))
        check(tree["maybeBig"] == JsonPrimitive(Long.MIN_VALUE.toString()))
        check(tree["bigList"]!!.jsonArray[0] == JsonPrimitive(Long.MAX_VALUE))
        check(tree["bigMap"]!!.jsonObject["large"] == JsonPrimitive(Long.MIN_VALUE))
        check(tree["child"]!!.jsonObject["big"] == JsonPrimitive("7"))
        check(tree["onlyNull"] === JsonNull && tree["days"]!!.jsonObject["missing"] === JsonNull)
        val back = codec.decodeFromJsonElement(SpecialInputDto.serializer(), tree)
        check(back.bytes.contentEquals(bytes) && back.big == Long.MAX_VALUE && back.maybeBig == Patch.Value(Long.MIN_VALUE))
        check(back.money == Money("12.50"))
        val absent = codec.encodeToJsonElement(SpecialInputDto.serializer(), SpecialInputDto(stamp, day, bytes, 1, JsonNull)).jsonObject
        check("maybeBig" !in absent)
        val explicit = codec.encodeToJsonElement(SpecialInputDto.serializer(), input.copy(maybeBig = Patch.Null)).jsonObject
        check(explicit["maybeBig"] === JsonNull)
        check(codec.decodeFromJsonElement(SpecialInputDto.serializer(), explicit).maybeBig === Patch.Null)
        for ((field, invalid) in listOf("stamp" to JsonPrimitive("not-an-instant"), "day" to JsonPrimitive("2024-02-30"),
            "bytes" to JsonPrimitive("base64-is-not-the-model-contract"), "big" to JsonPrimitive("9223372036854775808"),
            "onlyNull" to JsonPrimitive("not-null"))) {
            check(runCatching { codec.decodeFromJsonElement(SpecialInputDto.serializer(), JsonObject(tree + (field to invalid))) }.isFailure)
        }
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val payload = (request.body as OutgoingContent.ByteArrayContent).bytes()
            when (calls) {
                1 -> {
                    check(request.url.parameters["since"] == stamp.toString())
                    check(codec.parseToJsonElement(payload.decodeToString()) == tree)
                    respond("""{"stamp":"2024-02-29T12:34:56Z","day":"2024-02-29","bytes":[-128,-1,0,127],"big":"9223372036854775807","onlyNull":null,"nativeStamp":"2024-02-29T12:34:56Z"}""",
                        HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                2 -> {
                    check(request.url.encodedPath == "/raw-bytes" && payload.contentEquals(bytes))
                    respond(bytes, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                3 -> {
                    check(request.url.encodedPath == "/root-big" && payload.decodeToString() == Long.MAX_VALUE.toString())
                    respond(Long.MIN_VALUE.toString(), HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                }
                else -> error("Unexpected special scalar request")
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = ApiClient(http).default
            val record: RecordDto = api.special(since = stamp, input = input)
            check(record.stamp == stamp && record.day == day && record.bytes.contentEquals(bytes))
            check(record.big == Long.MAX_VALUE && record.onlyNull === JsonNull && record.nativeStamp == stamp)
            check(api.rawBytes(bytes).contentEquals(bytes))
            check(api.rootBig(Long.MAX_VALUE) == Long.MIN_VALUE)
            check(calls == 3)
            println("Special scalar contracts passed: defaults=$defaults/nulls=$nulls; Instant/LocalDate/ByteArray model vs raw body/bigint field vs numeric container/root/JsonNull/Patch; $calls requests")
        } finally { http.close(); engine.close() }
    }
}
