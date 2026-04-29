plugins {
    id("screentime.android.feature")
}

android {
    namespace = "com.nigel4321.screentime.feature.onboarding"
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom("$rootDir/config/detekt/detekt.yml")
}

dependencies {
    implementation(project(":core-ui"))
    implementation(project(":core-domain"))
}
