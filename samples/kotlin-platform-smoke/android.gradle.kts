plugins {
    id("com.android.library") version "8.13.2"
    kotlin("android") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}
repositories { google(); mavenCentral() }
val probePackage = providers.gradleProperty("androidProbePackage")
android {
    namespace = probePackage.orElse("org.example.platform").get()
    compileSdk = 36
    defaultConfig {
        minSdk = 23
        if (probePackage.isPresent) {
            testApplicationId = probePackage.get()
            testInstrumentationRunner = "org.example.platform.RoadmapProbeInstrumentation"
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    sourceSets["main"].java.srcDirs("generated", "generated-increments", "generated-roadmap", "generated-getters", "generated-enum-bounds", "src/commonMain/kotlin")
}
kotlin { compilerOptions.jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) }
dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0")
    implementation("org.jetbrains.kotlinx:kotlinx-datetime:0.7.1")
}
