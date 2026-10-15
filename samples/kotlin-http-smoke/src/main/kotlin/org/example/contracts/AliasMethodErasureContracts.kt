@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)

package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import org.example.aliasmethods.api.DefaultApi
import org.example.aliasmethods.models.AStringsDto
import org.example.aliasmethods.models.BIntsDto
import org.example.aliasmethods.models.MIntsDto
import org.example.aliasmethods.models.MStringsDto
import org.example.aliasmethods.models.ModelADto
import org.example.aliasmethods.models.ModelBDto
import org.example.aliasmethods.models.VStringADto
import org.example.aliasmethods.models.VStringBDto

private fun aliasMethodBody(body: OutgoingContent): String = when (body) {
    is OutgoingContent.ContentWrapper -> aliasMethodBody(body.delegate())
    is OutgoingContent.ByteArrayContent -> body.bytes().decodeToString()
    else -> error("Unexpected alias method body $body")
}

suspend fun aliasMethodErasureContracts() {
    val strings: AStringsDto = listOf("a")
    val ints: BIntsDto = listOf(2)
    val stringMap: MStringsDto = mapOf("k" to "v")
    val intMap: MIntsDto = mapOf("k" to 2)
    val expected = mapOf(
        "/a" to "[\"a\"]", "/b" to "[2]", "/c" to "[true]", "/d" to "null", "/z" to "[2]",
        "/maps/a" to "{\"k\":\"v\"}", "/maps/b" to "{\"k\":2}",
        "/models/a" to "{\"id\":\"a\"}", "/models/b" to "{\"id\":2}",
        "/wrapped/a" to "\"a\"", "/wrapped/b" to "\"b\"",
    )
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val seen = mutableListOf<String>()
        val engine = MockEngine { request ->
            val path = request.url.encodedPath
            check(request.method.value == "POST")
            check(aliasMethodBody(request.body) == expected.getValue(path)) { "Wrong alias method payload at $path" }
            seen += path
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = DefaultApi(http, "http://example.test/")
            api.send(strings)
            api.send3(ints)
            api.send4(listOf(true))
            api.send5()
            api.send2(ints) // Naturally declared send2 stays reserved, not stolen by a fallback.
            api.map(stringMap)
            api.map2(intMap)
            api.model(ModelADto("a"))
            api.model(ModelBDto(2))
            api.wrapped(VStringADto("a"))
            api.wrapped(VStringBDto("b"))
            check(seen.size == expected.size && seen.toSet() == expected.keys)
        } finally { http.close(); engine.close() }
        println("Alias method erasure contracts passed: defaults=$defaults/nulls=$nulls; List/Map/null/reserved suffix/nominal DTO-valueclass overloads")
    }
}
