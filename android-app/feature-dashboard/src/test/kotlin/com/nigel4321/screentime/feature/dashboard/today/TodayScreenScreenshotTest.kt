package com.nigel4321.screentime.feature.dashboard.today

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
import com.github.takahirom.roborazzi.RoborazziOptions
import com.github.takahirom.roborazzi.RoborazziRule
import com.github.takahirom.roborazzi.captureRoboImage
import com.nigel4321.screentime.core.domain.model.BundleId
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.core.ui.theme.ScreentimeTheme
import com.nigel4321.screentime.feature.dashboard.today.components.EmptyState
import com.nigel4321.screentime.feature.dashboard.today.components.ErrorState
import com.nigel4321.screentime.feature.dashboard.today.components.LoadingSkeleton
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import kotlin.time.Duration.Companion.minutes

/**
 * Renders each `TodayScreen` state as a PNG under
 * `build/outputs/roborazzi/` so a human can eyeball the layout without
 * an emulator. The android workflow uploads the directory as an
 * artifact called `today-screenshots-<sha>`.
 *
 * To regenerate locally (requires Android SDK):
 *   ./gradlew :feature-dashboard:recordRoborazziDebug
 *
 * The renderings here intentionally bypass the real `TodayViewModel`
 * and call the leaf components directly — we want predictable PNGs,
 * not a re-test of the ViewModel state machine (already covered by
 * `TodayViewModelTest`).
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(qualifiers = "w411dp-h891dp-xxhdpi")
class TodayScreenScreenshotTest {
    @get:Rule
    val composeRule = createComposeRule()

    @get:Rule
    val roborazziRule =
        RoborazziRule(
            composeRule = composeRule,
            captureRoot = composeRule.onRoot(),
            options =
                RoborazziRule.Options(
                    outputDirectoryPath = "build/outputs/roborazzi",
                ),
        )

    @Test
    fun loading() {
        composeRule.setContent {
            ScreentimeTheme {
                LoadingSkeleton(modifier = Modifier.fillMaxSize())
            }
        }
        composeRule.onRoot().captureRoboImage(
            filePath = "build/outputs/roborazzi/today_loading.png",
            roborazziOptions = RoborazziOptions(),
        )
    }

    @Test
    fun empty() {
        composeRule.setContent {
            ScreentimeTheme {
                EmptyState(modifier = Modifier.fillMaxSize())
            }
        }
        composeRule.onRoot().captureRoboImage(
            filePath = "build/outputs/roborazzi/today_empty.png",
        )
    }

    @Test
    fun error() {
        composeRule.setContent {
            ScreentimeTheme {
                ErrorState(
                    message = "Couldn't reach the backend",
                    onRetry = {},
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
        composeRule.onRoot().captureRoboImage(
            filePath = "build/outputs/roborazzi/today_error.png",
        )
    }

    @OptIn(ExperimentalMaterial3Api::class)
    @Test
    fun loaded() {
        composeRule.setContent {
            ScreentimeTheme {
                PullToRefreshBox(
                    isRefreshing = false,
                    onRefresh = {},
                    modifier = Modifier.fillMaxSize(),
                ) {
                    LoadedBentoSample()
                }
            }
        }
        composeRule.onRoot().captureRoboImage(
            filePath = "build/outputs/roborazzi/today_loaded.png",
        )
    }

    @Composable
    private fun LoadedBentoSample() {
        // Calls the same `internal` Composable production renders for
        // the Loaded state, so the PNG matches what users see.
        LoadedBento(
            rows =
                listOf(
                    UsageRow(BundleId("com.google.Chrome"), null, 92.minutes),
                    UsageRow(BundleId("com.tinyspeck.slackmacgap"), null, 51.minutes),
                    UsageRow(BundleId("com.apple.Terminal"), null, 38.minutes),
                    UsageRow(BundleId("com.spotify.client"), null, 24.minutes),
                    UsageRow(BundleId("com.apple.mail"), null, 11.minutes),
                ),
            total = 216.minutes,
        )
    }
}
