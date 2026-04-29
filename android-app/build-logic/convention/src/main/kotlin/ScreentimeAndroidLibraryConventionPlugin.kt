import com.android.build.api.dsl.LibraryExtension
import com.nigel4321.screentime.buildlogic.configureKotlinAndroid
import org.gradle.api.Plugin
import org.gradle.api.Project
import org.gradle.kotlin.dsl.configure

class ScreentimeAndroidLibraryConventionPlugin : Plugin<Project> {
    override fun apply(target: Project) {
        with(target) {
            with(pluginManager) {
                apply("com.android.library")
                apply("org.jetbrains.kotlin.android")
                apply("org.jlleitschuh.gradle.ktlint")
                apply("io.gitlab.arturbosch.detekt")
            }

            extensions.configure<LibraryExtension> {
                configureKotlinAndroid(this)
                defaultConfig.consumerProguardFiles("consumer-rules.pro")
            }
        }
    }
}
