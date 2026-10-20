@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import org.example.rawmedia.api.DefaultApi

public suspend fun rawResponseMediaContracts(checked: Boolean = false) {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val raw = "\"quoted\"\n雪 & + / %"
        val binary = byteArrayOf(0, 1, 127, -128, -1)
        var invalid = false
        val engine = MockEngine { request ->
            when (request.url.encodedPath) {
                "/text" -> respond(if (invalid) "" else raw, HttpStatusCode.OK, headersOf("Content-Type", "text/plain; charset=utf-8"))
                "/charset" -> respond(byteArrayOf(99, 97, 102, -23), HttpStatusCode.OK, headersOf("Content-Type", "text/plain; charset=ISO-8859-1"))
                "/empty" -> respond("", HttpStatusCode.OK, headersOf("Content-Type", "text/plain"))
                "/nullable-text" -> respond("null", HttpStatusCode.OK, headersOf("Content-Type", "text/plain"))
                "/bytes" -> respond(binary, HttpStatusCode.OK, headersOf("Content-Type", "application/octet-stream"))
                "/nullable-bytes" -> respond("null".encodeToByteArray(), HttpStatusCode.OK, headersOf("Content-Type", "application/octet-stream"))
                "/priority" -> respond("42", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
                "/raw-priority" -> respond(raw, HttpStatusCode.OK, headersOf("Content-Type", "text/plain"))
                "/status" -> respond("created", HttpStatusCode.Created, headersOf("Content-Type", "text/plain"))
                else -> error("Unexpected synthetic raw media path")
            }
        }
        val client = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = DefaultApi(client, "https://example.test/")
            check(api.readText() == raw)
            check(api.readCharset() == "café")
            check(api.readEmpty() == "")
            check(api.readNullableText() == "null")
            check(api.readBytes().contentEquals(binary))
            check(api.readNullableBytes()!!.contentEquals("null".encodeToByteArray()))
            check(api.readPriority() == 42)
            check(api.readRawPriority() == raw)
            check(api.readStatus() == "created")
            if (checked) {
                invalid = true
                check(runCatching { api.readText() }.isFailure)
                invalid = false
            }
            val bare = HttpClient(engine)
            try {
                val plainApi = DefaultApi(bare, "https://example.test/")
                check(plainApi.readText() == raw)
                check(plainApi.readBytes().contentEquals(binary))
            } finally { bare.close() }
        } finally { client.close(); engine.close() }
        println("Raw HTTP media defaults=$defaults nulls=$nulls text/charset/bytes/nullable/priorities/validation PASS")
    }
}
