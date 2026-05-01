import java.io.File

plugins {
    id("screentime.android.application.compose")
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.hilt)
    alias(libs.plugins.ksp)
}

// Release builds parse the version from the GITHUB_REF_NAME env var
// (the tag the release workflow checked out, e.g. `android-v0.2.5`).
// Local dev builds and any push that doesn't carry a tag fall back to
// the dev defaults below — the build still succeeds without env vars
// so contributors can `assembleDebug` without ceremony.
//
// versionCode is monotonic-by-construction: MAJOR*1_000_000 +
// MINOR*1_000 + PATCH lets us go up to 2.x.x without overflowing
// int32 and keeps the value readable when triaging Play Store crash
// reports.
val releaseVersionName: String =
    (System.getenv("GITHUB_REF_NAME") ?: "")
        .removePrefix("android-v")
        .ifBlank { "0.1.0-dev" }

val releaseVersionCode: Int =
    if (releaseVersionName == "0.1.0-dev") {
        1
    } else {
        val parts = releaseVersionName.split(".")
        require(parts.size == 3) { "android-v tag must be android-vMAJOR.MINOR.PATCH, got '$releaseVersionName'" }
        val (major, minor, patch) = parts.map { it.toInt() }
        major * 1_000_000 + minor * 1_000 + patch
    }

android {
    namespace = "com.nigel4321.macosscreentime"

    defaultConfig {
        applicationId = "com.nigel4321.macosscreentime"
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = releaseVersionCode
        versionName = releaseVersionName

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    // Release signing reads its inputs from env vars wired up by the
    // release workflow (which decodes a base64 keystore secret to a
    // tmp file and exports its path). When any required var is
    // missing — local dev, contributors without the keystore — the
    // signingConfig is skipped so `assembleRelease` still produces an
    // unsigned APK that can be inspected but won't install.
    val keystorePath: String? = System.getenv("RELEASE_KEYSTORE_PATH")
    val keystorePassword: String? = System.getenv("RELEASE_KEYSTORE_PASSWORD")
    val keyAlias: String? = System.getenv("RELEASE_KEY_ALIAS")
    val keyPassword: String? = System.getenv("RELEASE_KEY_PASSWORD")
    val hasReleaseSigning =
        keystorePath != null && keystorePassword != null && keyAlias != null && keyPassword != null

    signingConfigs {
        if (hasReleaseSigning) {
            create("release") {
                storeFile = File(keystorePath!!)
                storePassword = keystorePassword
                this.keyAlias = keyAlias
                this.keyPassword = keyPassword
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
            if (hasReleaseSigning) {
                signingConfig = signingConfigs.getByName("release")
            }
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
