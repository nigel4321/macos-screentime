plugins {
    id("screentime.android.library")
    alias(libs.plugins.hilt)
    alias(libs.plugins.ksp)
}

android {
    namespace = "com.nigel4321.screentime.core.data"
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom("$rootDir/config/detekt/detekt.yml")
}

dependencies {
    implementation(project(":core-domain"))

    implementation(libs.hilt.android)
    ksp(libs.hilt.compiler)
}
