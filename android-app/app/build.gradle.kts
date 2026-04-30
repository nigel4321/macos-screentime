plugins {
    id("screentime.android.application.compose")
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.hilt)
    alias(libs.plugins.ksp)
}

android {
    namespace = "com.nigel4321.macosscreentime"

    defaultConfig {
        applicationId = "com.nigel4321.macosscreentime"
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = 1
        versionName = "0.1.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    buildFeatures {
        buildConfig = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom("$rootDir/config/detekt/detekt.yml")
}

dependencies {
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.navigation.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    // Today + CalendarMonth icons in DashboardHost's bottom nav.
    // material-icons-core (transitive via material3) is too small.
    implementation(libs.androidx.compose.material.icons.extended)

    // DI — Hilt
    implementation(libs.hilt.android)
    ksp(libs.hilt.compiler)
    implementation(libs.hilt.navigation.compose)

    // Auth state observation drives the start destination.
    implementation(project(":core-data"))

    // Feature modules + design system
    implementation(project(":core-ui"))
    implementation(project(":core-domain"))
    implementation(project(":feature-onboarding"))
    implementation(project(":feature-dashboard"))
}
