package com.nigel4321.screentime.buildlogic

import com.android.build.api.dsl.CommonExtension
import org.gradle.api.Project
import org.gradle.api.artifacts.MinimalExternalModuleDependency
import org.gradle.api.artifacts.VersionCatalog
import org.gradle.kotlin.dsl.dependencies

internal fun Project.configureAndroidCompose(
    commonExtension: CommonExtension<*, *, *, *, *, *>,
) {
    commonExtension.apply {
        buildFeatures {
            compose = true
        }
    }

    dependencies {
        val bom = libs.requireLibrary("androidx-compose-bom")
        add("implementation", platform(bom))
        add("androidTestImplementation", platform(bom))
        add("implementation", libs.requireLibrary("androidx-compose-ui"))
        add("implementation", libs.requireLibrary("androidx-compose-ui-tooling-preview"))
        add("implementation", libs.requireLibrary("androidx-compose-material3"))
        add("debugImplementation", libs.requireLibrary("androidx-compose-ui-tooling"))
    }
}

private fun VersionCatalog.requireLibrary(alias: String): MinimalExternalModuleDependency =
    findLibrary(alias).get().get()
