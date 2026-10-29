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
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.serializer
import org.example.dictionarywrapper.RootDictionaryWrapperApiClient
import org.example.dictionarywrapper.api.models.SubmitInputDto
import org.example.dictionarywrapper.models.AInheritDto
import org.example.dictionarywrapper.models.AliasDto
import org.example.dictionarywrapper.models.BNarrowDto
import org.example.dictionarywrapper.models.CForceDto
import org.example.dictionarywrapper.models.EOpenDto
import org.example.dictionarywrapper.models.FEnumDto
import org.example.dictionarywrapper.models.GPeerDto
import org.example.dictionarywrapper.models.HolderDto
import org.example.dictionarywrapper.models.Patch
import org.example.dictionarywrapper.models.`DDefaultDto default`
import org.example.dictionarywrapper.models.`DOwnerAliasDto default`
import org.example.dictionarywrapper.models.`EOpenDto default`
import org.example.dictionarywrapper.models.`FEnumDto default`
import org.example.dictionarywrapper.models.`GPeerDto default`
import org.example.dictionarywrapper.models.`HNullableDto default`
import org.example.dictionarywrapper.models.`ZDefaultDto default`

private suspend fun dictionaryWrapperReject(message: String, block: suspend () -> Unit) {
    try {
        block()
        error("Missing dictionary wrapper rejection: $message")
    } catch (cause: Throwable) {
        check(generateSequence(cause) { it.cause }.any {
            (it is IllegalArgumentException || it is SerializationException) &&
                it.message?.contains(message, ignoreCase = true) == true
        }) { "Wrong dictionary wrapper failure: $cause" }
    }
}

private fun dictionaryWrapperBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> dictionaryWrapperBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected dictionary wrapper request body")
}

suspend fun rootDictionaryAllOfWrapperContracts(checked: Boolean) {
    check(serializer<AInheritDto>().descriptor.isNullable)
    check(!serializer<BNarrowDto>().descriptor.isNullable)
    check(!serializer<AliasDto>().descriptor.isNullable)
    check(serializer<CForceDto>().descriptor.isNullable)
    check(serializer<EOpenDto>().descriptor.isNullable)
    val map: AliasDto = mapOf("key" to "ok")
    val holder = HolderDto(required = map, nullable = null)
    check(holder.optional == null)
    check(`ZDefaultDto default`() == mapOf("source" to "value"))
    val wrapperDefault = `DDefaultDto default`()
    check(wrapperDefault == mapOf("wrapper" to "local"))
    check(`DOwnerAliasDto default`() == mapOf("alias" to "local"))
    check(wrapperDefault !== `DDefaultDto default`())
    check(`HNullableDto default`() == null)
    if (checked) {
        dictionaryWrapperReject("minProperties") { HolderDto(required = emptyMap(), nullable = null) }
        dictionaryWrapperReject("minLength") { HolderDto(required = mapOf("key" to "a"), nullable = null) }
    }
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.encodeToJsonElement(`EOpenDto default`()) == codec.parseToJsonElement("""{"wrapper":null}"""))
        check(codec.encodeToJsonElement(`FEnumDto default`()) == codec.parseToJsonElement("""{"enum":"one"}"""))
        check(codec.encodeToJsonElement(`GPeerDto default`()) == codec.parseToJsonElement("""{"peer":{"id":"ok"}}"""))
        check(codec.decodeFromString<AInheritDto>("null") == null)
        check(codec.decodeFromString<CForceDto>("null") == null)
        dictionaryWrapperReject("null") { codec.decodeFromString<AliasDto>("null") }
        check(codec.decodeFromString<AliasDto>(codec.encodeToString(map)) == map)
        check(codec.decodeFromString<EOpenDto>("null") == null)
        check(codec.encodeToJsonElement(codec.decodeFromString<FEnumDto>("""{"k":"one"}""")) ==
            codec.parseToJsonElement("""{"k":"one"}"""))
        check(codec.encodeToJsonElement(codec.decodeFromString<GPeerDto>("""{"k":{"id":"ok"}}""")) ==
            codec.parseToJsonElement("""{"k":{"id":"ok"}}"""))
        val modelJson = codec.encodeToJsonElement(holder)
        check(codec.encodeToJsonElement(codec.decodeFromString<HolderDto>(codec.encodeToString(holder))) == modelJson)
        check(codec.decodeFromString<HolderDto>("""{"required":{"key":"ok"}}""").optional == null)
        val absent = SubmitInputDto(required = map)
        val clear = SubmitInputDto(required = map, value = Patch.Null)
        val assigned = SubmitInputDto(required = map, value = Patch.Value(map))
        check("value" !in (codec.encodeToJsonElement(absent) as JsonObject))
        check((codec.encodeToJsonElement(clear) as JsonObject)["value"] == JsonNull)
        check((codec.encodeToJsonElement(assigned) as JsonObject)["value"] == codec.encodeToJsonElement(map))
        for (input in listOf(absent, clear, assigned)) {
            check(codec.encodeToJsonElement(codec.decodeFromString<SubmitInputDto>(codec.encodeToString(input))) ==
                codec.encodeToJsonElement(input))
        }
        if (checked) {
            val unchecked: AliasDto = codec.decodeFromString("""{"key":"a"}""")
            dictionaryWrapperReject("minLength") { HolderDto(required = unchecked, nullable = null) }
            dictionaryWrapperReject("minLength") { codec.decodeFromString<HolderDto>("""{"required":{"key":"a"}}""") }
            dictionaryWrapperReject("minProperties") { SubmitInputDto(required = map, value = Patch.Value(emptyMap())) }
        }
        var expected = ""
        var mode = 0
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            check(codec.parseToJsonElement(dictionaryWrapperBody(request.body)) == codec.parseToJsonElement(expected))
            if (request.url.encodedPath == "/patch") {
                respond("", HttpStatusCode.NoContent)
            } else {
                val response = when (mode) {
                    1 -> "null"
                    2 -> "{}"
                    3 -> """{"key":"a"}"""
                    else -> expected
                }
                respond(response, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = RootDictionaryWrapperApiClient(http, "https://example.test/").default
            expected = codec.encodeToString(map)
            check(api.echoMap(map) == map)
            mode = 1
            dictionaryWrapperReject("null") { api.echoMap(map) }
            if (checked) {
                val before = calls
                var callback = false
                dictionaryWrapperReject("minProperties") { api.echoMap(emptyMap()) { callback = true } }
                check(calls == before && !callback)
                mode = 2
                dictionaryWrapperReject("minProperties") { api.echoMap(map) }
                mode = 3
                dictionaryWrapperReject("minLength") { api.echoMap(map) }
            }
            mode = 0
            for (input in listOf(absent, clear, assigned)) {
                expected = codec.encodeToString(input)
                api.submit(input)
            }
        } finally {
            http.close()
        }
        println("Root dictionary allOf wrappers passed: checked=$checked defaults=$defaults nulls=$nulls")
    }
}
