plugins {
    id("screentime.android.feature")
}

android {
    namespace = "com.nigel4321.screentime.feature.onboarding"

    buildFeatures {
        buildConfig = true
    }

    defaultConfig {
        // The OAuth 2.0 Web Application client ID provisioned in the
        // Google Cloud project that owns the backend. Sourced from a
        // local file or environment variable; defaults to an empty
        // string so debug builds compile, but Credential Manager will
        // fail at runtime if it isn't set. See README for setup.
        val webClientId =
            providers.environmentVariable("SCREENTIME_GOOGLE_WEB_CLIENT_ID").orNull
                ?: providers.gradleProperty("screentime.googleWebClientId").orNull
                ?: ""
        buildConfigField(
            "String",
            "GOOGLE_WEB_CLIENT_ID",
            "\"$webClientId\"",
        )
    }
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom("$rootDir/config/detekt/detekt.yml")
}

dependencies {
    implementation(project(":core-ui"))
    implementation(project(":core-domain"))
    implementation(project(":core-data"))

    implementation(libs.androidx.credentials)
    implementation(libs.androidx.credentials.play.services.auth)
    implementation(libs.googleid)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.mockito.core)
}
