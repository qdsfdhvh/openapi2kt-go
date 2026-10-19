@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts

import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import org.example.nulldefaults.models.CountsDto
import org.example.nulldefaults.models.DayDto
import org.example.nulldefaults.models.HolderDto
import org.example.nulldefaults.models.LabelDto
import org.example.nulldefaults.models.ModeDto
import org.example.nulldefaults.models.NameDto
import org.example.nulldefaults.models.NamesDto
import org.example.nulldefaults.models.`CountsDto default`
import org.example.nulldefaults.models.`DayDto default`
import org.example.nulldefaults.models.`ModeDto default`
import org.example.nulldefaults.models.`NameDto default`
import org.example.nulldefaults.models.`NamesDto default`

// An explicitly nullable component root may carry a **non-null** default. The
// declaration keeps its nullable public type, so the factory returns a non-null
// instance of it: absent -> value, present null -> still null.
suspend fun nullableComponentDefaultsContracts() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }

        // Every admitted shape decodes its own annotation through the alias.
        check(`NameDto default`() == "fallback")
        check(`NamesDto default`() == listOf("a", "b"))
        check(`CountsDto default`() == mapOf("k" to 1))
        check(`DayDto default`() == kotlinx.datetime.LocalDate.parse("2024-02-29"))
        check(`ModeDto default`() == ModeDto.A)
        check(LabelDto.default()?.z == "label") { "nullable object factory lost its value" }

        // The nullable public type still accepts null, which no factory forces.
        val explicitNull: NameDto = null
        check(explicitNull == null)
        val nullNames: NamesDto = null
        check(nullNames == null)

        // Each call evaluates the factory again, so no caller shares one list.
        check(`NamesDto default`() !== `NamesDto default`())

        // A field referencing a defaulted component keeps its neutral presence.
        val holder = HolderDto()
        check(holder.name == null) { "Component default leaked into a referencing field" }
        check(holder.names == null) { "Component default leaked into a referencing field" }
        check(holder.inline == "inline")

        // The factory output round-trips through the same wire the fields use.
        val decoded = codec.decodeFromString<NameDto>(codec.encodeToString(`NameDto default`()))
        check(decoded == "fallback")
        check(codec.encodeToJsonElement(`NamesDto default`()) == codec.parseToJsonElement("""["a","b"]"""))
        check(codec.encodeToJsonElement(`CountsDto default`()) == codec.parseToJsonElement("""{"k":1}"""))
        check(codec.encodeToJsonElement(`DayDto default`()) == codec.parseToJsonElement("\"2024-02-29\""))
        check(codec.encodeToJsonElement(LabelDto.default()!!) == codec.parseToJsonElement("""{"z":"label"}"""))

        // Supplying null stays a distinct value from omitting the field.
        val explicit = codec.decodeFromString<HolderDto>("""{"name":null,"names":["c"]}""")
        check(explicit.name == null && explicit.names == listOf("c"))
        val supplied = codec.decodeFromString<HolderDto>("""{"name":"given"}""")
        check(supplied.name == "given")

        println("NullableComponentDefaults Json defaults=$defaults nulls=$nulls passed")
    }
}
