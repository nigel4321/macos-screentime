package com.nigel4321.screentime.feature.dashboard.week

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithText
import com.nigel4321.screentime.core.domain.model.BundleId
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.core.ui.theme.ScreentimeTheme
import com.nigel4321.screentime.feature.dashboard.today.components.EmptyState
import com.nigel4321.screentime.feature.dashboard.today.components.ErrorState
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode
import java.time.LocalDate
import kotlin.time.Duration.Companion.minutes

@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(qualifiers = "w411dp-h891dp-xxhdpi")
class WeekScreenSemanticsTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun empty_uses_the_shared_today_copy() {
        composeRule.setContent {
            ScreentimeTheme { EmptyState(modifier = Modifier.fillMaxSize()) }
        }
        // Week reuses the shared EmptyState — same copy applies, since
        // the Mac agent isn't recording anything regardless of which
        // tab the user is on.
        composeRule.onNodeWithText("No usage today yet").assertIsDisplayed()
    }

    @Test
    fun error_shows_message_and_retry() {
        composeRule.setContent {
            ScreentimeTheme {
                ErrorState(
                    message = "Backend unreachable",
                    onRetry = {},
                    modifier = Modifier.fillMaxSize(),
                )
            }
        }
        composeRule.onNodeWithText("Couldn't load today").assertIsDisplayed()
        composeRule.onNodeWithText("Retry").assertIsDisplayed()
    }

    @Test
    fun loaded_renders_total_chart_and_top_apps() {
        val days =
            (0..6).map { offset ->
                DayBucket(
                    day = LocalDate.of(2026, 4, 24).plusDays(offset.toLong()),
                    duration = (10 * (offset + 1)).minutes,
                )
            }
        val topApps =
            listOf(
                UsageRow(BundleId("com.a"), null, 90.minutes, displayName = "App A"),
                UsageRow(BundleId("com.b"), null, 50.minutes),
            )
        composeRule.setContent {
            ScreentimeTheme {
                LoadedBento(byDay = days, topApps = topApps, total = 280.minutes)
            }
        }
        composeRule.onNodeWithText("Total this week").assertIsDisplayed()
        composeRule.onNodeWithText("4h 40m").assertIsDisplayed()
        composeRule.onNodeWithText("Daily totals").assertIsDisplayed()
        composeRule.onNodeWithText("App A").assertIsDisplayed()
        composeRule.onNodeWithText("com.b").assertIsDisplayed()
    }
}
