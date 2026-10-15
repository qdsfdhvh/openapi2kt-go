package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.HttpStatusCode
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import org.example.exclusions.MergeExclusionsApiClient
import org.example.exclusions.models.AlphaDto
import org.example.exclusions.models.MiddleDto
import org.example.exclusions.models.ZetaDto
import org.example.exclusions.models.KeepADtoDto
import org.example.exclusions.models.KeepZDtoDto

private fun exclusionBody(content: OutgoingContent): String = when (content) {
    is OutgoingContent.ContentWrapper -> exclusionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic body")
}

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun mergeExclusionContracts() {
    val rootName = "org.example.exclusions.models.AlphaDto"
    val patterns = listOf(
        "org\\.example\\.exclusions\\.models\\.(Alpha|Zeta)Dto" to true,
        "AlphaDto" to false,
        "[A-Za-z0-9_.]+" to true,
        "^org\\.example\\.exclusions\\.models\\.AlphaDto$" to true,
        ".*\\.AlphaDto{1,2}" to true,
        "org.*ZetaDto" to false
    )
    for ((pattern, expected) in patterns) check(Regex(pattern).matches(rootName) == expected)
    val root = AlphaDto("ready")
    val alias: MiddleDto = root
    val protected = ZetaDto("ready")
    val keptA = KeepADtoDto("ready")
    val keptZ = KeepZDtoDto("ready")
    check(root == alias)
    check(serializer<MiddleDto>().descriptor.serialName == serializer<AlphaDto>().descriptor.serialName)
    check(serializer<ZetaDto>().descriptor.serialName != serializer<AlphaDto>().descriptor.serialName)
    check(serializer<KeepADtoDto>().descriptor.serialName != serializer<KeepZDtoDto>().descriptor.serialName)
    check((protected as Any) != root && (keptA as Any) != keptZ)
    check(runCatching { ZetaDto("") }.isFailure)
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        check(codec.encodeToString(root) == codec.encodeToString(protected))
        check(codec.decodeFromString<ZetaDto>(codec.encodeToString(protected)) == protected)
        check(codec.decodeFromString<KeepZDtoDto>(codec.encodeToString(keptZ)) == keptZ)
        check(runCatching { codec.decodeFromString<ZetaDto>("{\"value\":\"\"}") }.isFailure)
        val http = HttpClient(MockEngine { request ->
            if (request.url.encodedPath == "/write") check(codec.decodeFromString<KeepZDtoDto>(exclusionBody(request.body)) == keptZ)
            respond("{\"value\":\"ready\"}", HttpStatusCode.OK, headersOf("Content-Type", "application/json"))
        }) { install(ContentNegotiation) { json(codec) } }
        try {
            val api = MergeExclusionsApiClient(http, "http://example.test/").default
            check(api.read() == protected)
            check(api.write(keptZ) == keptZ)
            println("MergeExclusions Json defaults=$defaults nulls=$nulls full-match-suffix-identities-codecs-HTTP passed")
        } finally { http.close() }
    }
}
