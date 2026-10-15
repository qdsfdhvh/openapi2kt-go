package org.example.contracts

import org.example.plainprobe.models.SampleDto as DataSample
import org.example.actualplain.models.SampleDto as PlainSample
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement

private fun plain(source: DataSample): PlainSample = PlainSample(source.flag,source.single,source.wide,source.bytes,source.providerBytes,source.values,source.entries,source.optionalSingle,source.optionalWide,source.optionalBytes)
fun actualPlainStyleContracts(){
 val bytes=byteArrayOf(-128,0,127)
 val samples=mutableListOf<DataSample>()
 for(flag in listOf(false,true)) for(f in listOf(0f,-0f,1f,Float.NaN,Float.fromBits(0x7fc00001),Float.POSITIVE_INFINITY)) for(d in listOf(0.0,-0.0,1.0,Double.NaN,Double.fromBits(0x7ff8000000000001L),Double.NEGATIVE_INFINITY)) for(nullable in listOf(false,true)){
 samples+=DataSample(flag,f,d,bytes,bytes,listOf(0.0,-0.0,Double.NaN),mapOf("sample" to Long.MIN_VALUE),if(nullable)null else f,if(nullable)null else d,if(nullable)null else bytes)
 }
 for(a in samples){ val p=plain(a);check(p.hashCode()==a.hashCode());check(p.toString()=="SampleDto");check(p==p);check(p!=null);check(p!=plain(a.copy(bytes=bytes.copyOf())));check(p.hashCode()==plain(a.copy(bytes=bytes.copyOf())).hashCode());for(b in samples)check((a==b)==(p==plain(b))) }
 check(PlainSample::class.java.methods.none{it.name=="copy"||it.name.startsWith("component")})
 check(DataSample::class.java.methods.any{it.name=="copy"})
 val finite=samples.first().copy(single=1.5f,wide=-2.5,values=listOf(0.5),optionalSingle=null,optionalWide=-0.0)
 for(defaults in listOf(false,true))for(nulls in listOf(false,true)){
 val json=Json{encodeDefaults=defaults;explicitNulls=nulls};val dataWire=json.encodeToJsonElement(finite);val plainWire=json.encodeToJsonElement(plain(finite));check(plainWire==dataWire)
 val decoded=json.decodeFromJsonElement(PlainSample.serializer(),plainWire);check(json.encodeToJsonElement(decoded)==plainWire);check(decoded.bytes.contentEquals(bytes)&&decoded.providerBytes.contentEquals(bytes))
 println("ActualPlainStyle Json defaults=$defaults nulls=$nulls generated-floating-providerArray-equals-hash-no-copy-componentN-wire checks passed")
 }
 println("ACTUAL PLAIN STYLE 144 GENERATED SAMPLES/20736 EQUALITY PAIRS PASS")
}
