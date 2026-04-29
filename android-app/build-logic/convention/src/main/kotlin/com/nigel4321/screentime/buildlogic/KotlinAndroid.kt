package com.nigel4321.screentime.buildlogic

import com.android.build.api.dsl.CommonExtension
import org.gradle.api.JavaVersion
import org.gradle.api.Project
import org.gradle.kotlin.dsl.configure
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.dsl.KotlinAndroidProjectExtension

internal fun Project.configureKotlinAndroid(
    commonExtension: CommonExtension<*, *, *, *, *, *>,
) {
    commonExtension.apply {
        compileSdk = libs.intVersion("compileSdk")

        defaultConfig {
            minSdk = libs.intVersion("minSdk")
        }

        compileOptions {
            sourceCompatibility = JavaVersion.VERSION_21
            targetCompatibility = JavaVersion.VERSION_21
        }
    }

    configure<KotlinAndroidProjectExtension> {
        jvmToolchain(libs.intVersion("java"))
        compilerOptions {
            jvmTarget.set(JvmTarget.JVM_21)
        }
    }
}
