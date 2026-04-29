import com.android.build.api.dsl.ApplicationExtension
import com.nigel4321.screentime.buildlogic.configureKotlinAndroid
import com.nigel4321.screentime.buildlogic.intVersion
import com.nigel4321.screentime.buildlogic.libs
import org.gradle.api.Plugin
import org.gradle.api.Project
import org.gradle.kotlin.dsl.configure

class ScreentimeAndroidApplicationConventionPlugin : Plugin<Project> {
    override fun apply(target: Project) {
        with(target) {
            with(pluginManager) {
                apply("com.android.application")
                apply("org.jetbrains.kotlin.android")
                apply("org.jlleitschuh.gradle.ktlint")
                apply("io.gitlab.arturbosch.detekt")
            }

            extensions.configure<ApplicationExtension> {
                configureKotlinAndroid(this)
                defaultConfig.targetSdk = libs.intVersion("targetSdk")
            }
        }
    }
}
