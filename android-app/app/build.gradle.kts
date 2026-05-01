import java.io.File

plugins {
    id("screentime.android.application.compose")
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.hilt)
    alias(libs.plugins.ksp)
}

// Release builds parse the version from the GITHUB_REF_NAME env var
// when (and only when) it carries the `android-v` tag prefix the
// release workflow uses. Anything else — branch-name pushes, PR
// builds, local dev — falls back to the dev defaults below so
// `assembleDebug` keeps working without ceremony.
//
// versionCode is monotonic-by-construction: MAJOR*1_000_000 +
// MINOR*1_000 + PATCH lets us go up to 2.x.x without overflowing
// int32 and keeps the value readable when triaging Play Store crash
// reports.
val androidTagPattern = Regex("""^android-v(\d+)\.(\d+)\.(\d+)$""")

val (releaseVersionName: String, releaseVersionCode: Int) =
    androidTagPattern.matchEntire(System.getenv("GITHUB_REF_NAME") ?: "")
        ?.destructured
        ?.let { (major, minor, patch) ->
            "$major.$minor.$patch" to (major.toInt() * 1_000_000 + minor.toInt() * 1_000 + patch.toInt())
        }
        ?: ("0.1.0-dev" to 1)

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
