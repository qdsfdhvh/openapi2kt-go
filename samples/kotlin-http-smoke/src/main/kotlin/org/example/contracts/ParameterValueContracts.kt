package org.example.contracts

import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.http.HttpStatusCode
import io.ktor.http.decodeURLPart
import kotlinx.serialization.json.*
import org.example.parametervalues.ApiClient
import org.example.parametervalues.api.models.FilterValuesFilterDto
import org.example.parametervalues.api.models.ParameterValuesInlineStatesDto
import org.example.parametervalues.models.RecordDto
import org.example.parametervalues.models.StateDto

@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
suspend fun parameterValueContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
        var calls = 0
        val engine = MockEngine { request ->
            calls++
            val q = request.url.parameters
            when (calls) {
                1 -> {
                    val path = request.url.encodedPath.removePrefix("/collections/")
                    check(path.decodeURLPart() == "a,null,b/x" && !path.contains('/'))
                    check(q.getAll("repeat") == listOf("a", "b"))
                    check(q["csv"] == "a,null,b")
                    check(q["indexed[0]"] == "1" && q["indexed[1]"] == null && q["indexed[2]"] == "3")
                    check(q["nested"] == "[a, b],[c]")
                    check(q["kept"] == "7" && "gone" !in q.names())
                    check(q.getAll("states") == listOf("wire-ready", "future"))
                    check(q["stateCsv"] == "wire-ready,null,future")
                    check(q.getAll("inlineStates") == listOf("inline-ready", "future-inline"))
                    check(q["json"] == "\"quoted\"")
                    // Map entries are passed directly to Ktor, not enum-unwrapped.
                    check(q["status"] == "WIRE_READY")
                    check(q["record"] == "RecordDto(v=object)")
                    check(q["label"] == "\"quoted\"" && "absent" !in q.names())
                    check(request.headers["X-Values"] == "true,null,false")
                    check(request.headers["X-Map"] == "{a=1, b=2}")
                }
                2 -> check(q.isEmpty() && request.headers["X-Values"] == null)
                3 -> {
                    check(q.names() == setOf("csv", "nested"))
                    check(q["csv"] == "" && q["nested"] == "")
                    check(request.headers["X-Values"] == "")
                }
                4 -> {
                    check(request.url.encodedPath == "/filter")
                    check(q["filter[tags]"] == "[a, b]")
                    check(q["filter[child]"] == "Child(v=object)")
                    check(q["filter[map]"] == "{a=1, b=null}")
                    check(q["filter[opaque]"] == "\"quoted\"")
                    check(q["filter[nullableOpaque]"] == "null")
                    check(q.names().size == 5)
                }
                5 -> check(q.isEmpty())
                6 -> {
                    check(q.names() == setOf("filter[tags]", "filter[map]"))
                    check(q["filter[tags]"] == "[]" && q["filter[map]"] == "{}")
                }
                else -> error("Unexpected parameter value request")
            }
            respond("", HttpStatusCode.NoContent)
        }
        val http = HttpClient(engine)
        try {
            val api = ApiClient(http).default
            val ready = StateDto.fromSerialName("wire-ready")
            val future = StateDto.fromSerialName("future")
            api.parameterValues(
                pathList = listOf("a", null, "b/x"), repeat = listOf("a", null, "b"), csv = listOf("a", null, "b"),
                indexed = listOf(1, null, 3), nested = listOf(listOf("a", "b"), listOf("c")),
                map = linkedMapOf("kept" to 7, "gone" to null), states = listOf(ready, null, future),
                stateCsv = listOf(ready, null, future),
                inlineStates = listOf(ParameterValuesInlineStatesDto.fromSerialName("inline-ready"), null,
                    ParameterValuesInlineStatesDto.fromSerialName("future-inline")),
                json = JsonPrimitive("quoted"), stateMap = mapOf("status" to ready), record = RecordDto("object"),
                raw = mapOf("label" to JsonPrimitive("quoted"), "absent" to null),
                xValues = listOf(true, null, false), xMap = linkedMapOf("a" to 1, "b" to 2),
            )
            api.parameterValues(listOf("single"))
            api.parameterValues(listOf("single"), repeat = emptyList(), csv = emptyList(), nested = emptyList(),
                indexed = emptyList(), map = emptyMap(), states = emptyList(), xValues = emptyList())
            val filter = FilterValuesFilterDto(tags = listOf("a", "b"), child = FilterValuesFilterDto.Child("object"),
                map = linkedMapOf("a" to 1, "b" to null), opaque = JsonPrimitive("quoted"), nullableOpaque = JsonNull)
            val roundTrip = codec.decodeFromJsonElement(FilterValuesFilterDto.serializer(), codec.encodeToJsonElement(FilterValuesFilterDto.serializer(), filter))
            check(roundTrip.opaque == JsonPrimitive("quoted"))
            api.filterValues(filter)
            api.filterValues(null)
            api.filterValues(FilterValuesFilterDto(tags = emptyList(), map = emptyMap()))
            check(calls == 6)
            println("Parameter value contracts passed: defaults=$defaults/nulls=$nulls; nullable elements/enum safe access, nested lists, maps/named/raw values and filter toString/null omission; $calls requests")
        } finally {
            http.close()
            engine.close()
        }
    }
}
