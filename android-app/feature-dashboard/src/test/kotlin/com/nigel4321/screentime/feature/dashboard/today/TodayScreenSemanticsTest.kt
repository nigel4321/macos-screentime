package com.nigel4321.screentime.feature.dashboard.today

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.ui.Modifier
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createComposeRule
import androidx.compose.ui.test.onNodeWithContentDescription
import androidx.compose.ui.test.onNodeWithText
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
 * Functional Compose UI tests with semantic assertions — closes the
 * §2.17 deferral. Pairs with the Roborazzi screenshot tests: those
 * catch visual regressions, these catch broken state-to-UI wiring.
 */
@RunWith(RobolectricTestRunner::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(qualifiers = "w411dp-h891dp-xxhdpi")
class TodayScreenSemanticsTest {
    @get:Rule
    val composeRule = createComposeRule()

    @Test
    fun loading_renders_loading_today_semantics() {
        composeRule.setContent {
            ScreentimeTheme { LoadingSkeleton(modifier = Modifier.fillMaxSize()) }
        }
        composeRule
            .onNodeWithContentDescription("Loading today")
            .assertIsDisplayed()
    }

    @Test
    fun empty_renders_no_usage_copy() {
        composeRule.setContent {
            ScreentimeTheme { EmptyState(modifier = Modifier.fillMaxSize()) }
        }
        composeRule.onNodeWithText("No usage today yet").assertIsDisplayed()
        composeRule.onNodeWithText("Open an app on the Mac to record activity.").assertIsDisplayed()
    }

    @Test
    fun error_renders_message_and_retry_button() {
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
        composeRule.onNodeWithText("Backend unreachable").assertIsDisplayed()
        composeRule.onNodeWithText("Retry").assertIsDisplayed()
    }

    @Test
    fun loaded_renders_total_top_apps_and_placeholder_tiles() {
        val rows =
            listOf(
                UsageRow(BundleId("com.a"), null, 30.minutes, displayName = "App A"),
                UsageRow(BundleId("com.b"), null, 15.minutes),
            )
        composeRule.setContent {
            ScreentimeTheme { LoadedBento(rows = rows, total = 45.minutes) }
        }
        composeRule.onNodeWithText("Total today").assertIsDisplayed()
        composeRule.onNodeWithText("45m").assertIsDisplayed()
        // displayName takes precedence; bundle-id-only row falls back.
        composeRule.onNodeWithText("App A").assertIsDisplayed()
        composeRule.onNodeWithText("com.b").assertIsDisplayed()
        // Categories + Downtime placeholders both anchor the layout
        // until §3.7 / §4.1 fill them.
        composeRule.onNodeWithText("Categories").assertIsDisplayed()
        composeRule.onNodeWithText("Downtime").assertIsDisplayed()
    }
}
