plugins {
    id("screentime.android.feature")
}

android {
    namespace = "com.nigel4321.screentime.feature.dashboard"

    testOptions {
        unitTests {
            isIncludeAndroidResources = true
        }
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

    implementation(libs.vico.compose.m3)

    testImplementation(libs.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.robolectric)
    testImplementation(libs.androidx.test.core)
}
