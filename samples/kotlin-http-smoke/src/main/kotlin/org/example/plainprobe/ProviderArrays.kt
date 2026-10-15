package org.example.plainprobe

import kotlinx.serialization.KSerializer
import kotlinx.serialization.Serializable
import kotlinx.serialization.descriptors.PrimitiveKind
import kotlinx.serialization.descriptors.PrimitiveSerialDescriptor
import kotlinx.serialization.descriptors.SerialDescriptor
import kotlinx.serialization.encoding.Decoder
import kotlinx.serialization.encoding.Encoder

public typealias ProviderShorts = ShortArray
public typealias ProviderInts = IntArray
public typealias ProviderLongs = LongArray
public typealias ProviderSingles = FloatArray
public typealias ProviderDoubles = DoubleArray
public typealias ProviderChars = CharArray
public typealias ProviderFlags = BooleanArray
public typealias ProviderItems = Array<String?>
public typealias ProviderAny = @Serializable(with = ProviderAnySerializer::class) Any
// Synthetic codec owns opaque values; the generator must not inspect the value.
public object ProviderAnySerializer : KSerializer<Any> {
    override val descriptor: SerialDescriptor = PrimitiveSerialDescriptor("SyntheticOpaque", PrimitiveKind.STRING)
    override fun serialize(encoder: Encoder, value: Any) { encoder.encodeString("opaque") }
    override fun deserialize(decoder: Decoder): Any = decoder.decodeString()
}
