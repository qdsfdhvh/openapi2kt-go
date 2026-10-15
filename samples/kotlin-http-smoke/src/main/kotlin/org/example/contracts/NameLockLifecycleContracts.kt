@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.serialization.kotlinx.json.json
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.KSerializer
import kotlinx.serialization.json.Json

// Main.kt supplies compiled call sites using classifier names from the migration
// report. Each phase has its own fresh generated sources and Gradle project.
data class NameLockLifecycleProbe<T : Any, M : Any>(
    val response: () -> T,
    val peerResponse: () -> T,
    val responseCodec: KSerializer<T>,
    val peerResponseCodec: KSerializer<T>,
    val canonicalResponseCodec: KSerializer<T>,
    val responseClasses: List<Class<*>>,
    val responseDescriptor: String,
    val nestedCodecs: List<Pair<KSerializer<*>, String>>,
    val modeFromSerialName: (String) -> M,
    val peerModeFromSerialName: (String) -> M,
    val modeSerialName: (M) -> String,
    val modeCodec: KSerializer<M>,
    val peerModeCodec: KSerializer<M>,
    val modeClasses: List<Class<*>>,
    val modeDescriptor: String,
    val paths: List<String>,
    val invoke: suspend (HttpClient, M) -> List<T>,
)

suspend fun <T : Any, M : Any> nameLockLifecycleContracts(
    phase: String,
    plain: Boolean,
    probe: NameLockLifecycleProbe<T, M>,
) = withTimeout(15000) {
    val response = probe.response()
    val peer = probe.peerResponse()
    check(response == peer && response.hashCode() == peer.hashCode())
    check(probe.responseClasses.all { it === response.javaClass })
    check(response.javaClass.name == probe.responseDescriptor)
    val hasCopy = response.javaClass.methods.any { it.name == "copy" }
    val hasComponent = response.javaClass.methods.any { it.name == "component1" }
    check(hasCopy == !plain && hasComponent == !plain)
    if (plain) check(response.toString() == probe.responseDescriptor.substringAfterLast('.'))
    for (codec in listOf(probe.responseCodec, probe.peerResponseCodec, probe.canonicalResponseCodec)) {
        check(codec.descriptor.serialName == probe.responseDescriptor)
    }
    for ((codec, expected) in probe.nestedCodecs) {
        check(codec.descriptor.serialName == expected)
    }

    val wire = """{"id":"record","child":{"status":"future","detail":{"enabled":true}}}"""
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        val expected = codec.parseToJsonElement(wire)
        for (serializer in listOf(probe.responseCodec, probe.peerResponseCodec, probe.canonicalResponseCodec)) {
            check(codec.parseToJsonElement(codec.encodeToString(serializer, response)) == expected)
            check(codec.decodeFromString(serializer, wire) == peer)
        }
        for (name in listOf("fast", "safe", "turbo")) {
            // Known and unknown values remain callable through the exported
            // aliases; sealed enum objects require the canonical nested owner.
            val mode = probe.modeFromSerialName(name)
            val peerMode = probe.peerModeFromSerialName(name)
            check(mode == peerMode)
            check(probe.modeSerialName(mode) == name)
            check(probe.modeClasses.all { it.isInstance(mode) })
            for (serializer in listOf(probe.modeCodec, probe.peerModeCodec)) {
                check(serializer.descriptor.serialName == probe.modeDescriptor)
                check(codec.encodeToString(serializer, mode) == "\"$name\"")
                check(codec.decodeFromString(serializer, "\"$name\"") == mode)
            }
            val seen = mutableListOf<String>()
            val engine = MockEngine { request ->
                check(request.method.value == "GET")
                check(request.url.host == "example.test")
                check(request.url.parameters.names() == setOf("mode"))
                check(request.url.parameters["mode"] == name)
                seen.add(request.url.encodedPath)
                respond(wire, HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
            }
            val http = HttpClient(engine) { install(ContentNegotiation) { json(codec) } }
            try {
                check(probe.invoke(http, mode) == List(probe.paths.size) { response })
                check(seen == probe.paths)
            } finally {
                http.close()
                engine.close()
            }
        }
        println("NameLockLifecycle phase=$phase plain=$plain defaults=$defaults nulls=$nulls aliases-descriptors-nested-enum-wire-HTTP PASS")
    }
}
