plugins {
    `kotlin-dsl`
}

group = "com.nigel4321.screentime.buildlogic"

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(libs.versions.java.get().toInt())
    }
}

dependencies {
    compileOnly(libs.android.gradlePlugin)
    compileOnly(libs.kotlin.gradlePlugin)
    compileOnly(libs.compose.gradlePlugin)
    compileOnly(libs.ksp.gradlePlugin)
}

gradlePlugin {
    plugins {
        register("androidApplication") {
            id = "screentime.android.application"
            implementationClass = "ScreentimeAndroidApplicationConventionPlugin"
        }
        register("androidApplicationCompose") {
            id = "screentime.android.application.compose"
            implementationClass = "ScreentimeAndroidApplicationComposeConventionPlugin"
        }
        register("androidLibrary") {
            id = "screentime.android.library"
            implementationClass = "ScreentimeAndroidLibraryConventionPlugin"
        }
        register("androidLibraryCompose") {
            id = "screentime.android.library.compose"
            implementationClass = "ScreentimeAndroidLibraryComposeConventionPlugin"
        }
        register("androidFeature") {
            id = "screentime.android.feature"
            implementationClass = "ScreentimeAndroidFeatureConventionPlugin"
        }
        register("kotlinLibrary") {
            id = "screentime.kotlin.library"
            implementationClass = "ScreentimeKotlinLibraryConventionPlugin"
        }
    }
}
