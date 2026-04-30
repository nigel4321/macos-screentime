plugins {
    id("screentime.android.feature")
    alias(libs.plugins.roborazzi)
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

    // TodayViewModelTest stands up its own Retrofit instance against a
    // MockWebServer; :core-data uses `implementation` for these, so we
    // have to re-declare them on the test classpath.
    testImplementation(libs.retrofit)
    testImplementation(libs.retrofit.kotlinx.serialization.converter)
    testImplementation(libs.kotlinx.serialization.json)

    // Compose UI test + Roborazzi screenshot capture for `TodayScreen`
    // states. Renders Composables to PNG under
    // build/outputs/roborazzi/ via `recordRoborazziDebug`. CI uploads
    // those PNGs as a workflow artifact for human inspection.
    val composeBom = platform(libs.androidx.compose.bom)
    testImplementation(composeBom)
    testImplementation(libs.androidx.compose.ui.test.junit4)
    debugImplementation(libs.androidx.compose.ui.test.manifest)
    testImplementation(libs.roborazzi)
    testImplementation(libs.roborazzi.compose)
    testImplementation(libs.roborazzi.junit.rule)
}
