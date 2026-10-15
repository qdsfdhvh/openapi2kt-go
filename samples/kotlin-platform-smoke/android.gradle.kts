plugins {
    id("com.android.library") version "8.13.2"
    kotlin("android") version "2.4.10"
    kotlin("plugin.serialization") version "2.4.10"
}
repositories { google(); mavenCentral() }
android {
    namespace = "org.example.platform"
    compileSdk = 36
    defaultConfig { minSdk = 23 }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    sourceSets["main"].java.srcDirs("generated", "src/commonMain/kotlin")
}
kotlin { compilerOptions.jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17) }
dependencies { implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.11.0") }
