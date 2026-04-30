package com.nigel4321.screentime.feature.dashboard.week

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onRoot
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
import java.time.LocalDate
import kotlin.time.Duration.Companion.minutes

/**
 * Renders each `WeekScreen` state to PNG under
 * `build/outputs/roborazzi/`. The android workflow uploads the
 * directory as `today-screenshots-<sha>` (the artifact is shared with
 * the Today screenshots since they live in the same module's output
 * directory).
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(qualifiers = "w411dp-h891dp-xxhdpi")
class WeekScreenScreenshotTest {
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
            ScreentimeTheme { LoadingSkeleton(modifier = Modifier.fillMaxSize()) }
        }
        composeRule.onRoot().captureRoboImage(filePath = "build/outputs/roborazzi/week_loading.png")
    }

    @Test
    fun empty() {
        composeRule.setContent {
            ScreentimeTheme { EmptyState(modifier = Modifier.fillMaxSize()) }
        }
        composeRule.onRoot().captureRoboImage(filePath = "build/outputs/roborazzi/week_empty.png")
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
        composeRule.onRoot().captureRoboImage(filePath = "build/outputs/roborazzi/week_error.png")
    }

    @Test
    fun loaded() {
        // Sample week ending today (clock-agnostic; the screenshot is
        // labelled "Mon Tue Wed …" by the chart helper). Durations
        // ramp up across the week and one day is zero so the empty
        // bar's track shows.
        val today = LocalDate.of(2026, 4, 30)
        val days =
            listOf(
                DayBucket(today.minusDays(6), 60.minutes),
                DayBucket(today.minusDays(5), 90.minutes),
                // Empty day — chart should show only the grey track here.
                DayBucket(today.minusDays(4), 0.minutes),
                DayBucket(today.minusDays(3), 120.minutes),
                DayBucket(today.minusDays(2), 75.minutes),
                DayBucket(today.minusDays(1), 200.minutes),
                DayBucket(today, 145.minutes),
            )
        val topApps =
            listOf(
                UsageRow(BundleId("com.google.Chrome"), null, 320.minutes, displayName = "Google Chrome"),
                UsageRow(BundleId("com.tinyspeck.slackmacgap"), null, 180.minutes, displayName = "Slack"),
                UsageRow(BundleId("com.apple.Terminal"), null, 95.minutes, displayName = null),
                UsageRow(BundleId("com.spotify.client"), null, 60.minutes, displayName = "Spotify"),
            )
        composeRule.setContent {
            ScreentimeTheme {
                LoadedBento(byDay = days, topApps = topApps, total = 690.minutes)
            }
        }
        composeRule.onRoot().captureRoboImage(filePath = "build/outputs/roborazzi/week_loaded.png")
    }
}
