plugins {
    id("screentime.android.library.compose")
}

android {
    namespace = "com.nigel4321.screentime.core.ui"
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom("$rootDir/config/detekt/detekt.yml")
}
