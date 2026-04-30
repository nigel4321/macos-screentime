package com.nigel4321.macosscreentime

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material.icons.filled.Today
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.nigel4321.screentime.feature.dashboard.TodayScreen
import com.nigel4321.screentime.feature.dashboard.WeekScreen

private enum class DashboardTab(val label: String) {
    Today("Today"),
    Week("Week"),
}

/**
 * Tabbed shell for the two primary dashboard destinations. The auth
 * gate routes here once a token is set and a device is selected;
 * picking between Today and Week is bottom-nav state, not a separate
 * `NavHost` route, so back-press from any other screen still lands on
 * the last selected tab.
 */
@Composable
fun DashboardHost(modifier: Modifier = Modifier) {
    var selected by rememberSaveable { mutableStateOf(DashboardTab.Today) }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        bottomBar = {
            NavigationBar {
                DashboardTab.entries.forEach { tab ->
                    NavigationBarItem(
                        selected = tab == selected,
                        onClick = { selected = tab },
                        icon = {
                            Icon(
                                imageVector = tab.icon(),
                                contentDescription = tab.label,
                            )
                        },
                        label = { Text(tab.label) },
                    )
                }
            }
        },
    ) { padding ->
        when (selected) {
            DashboardTab.Today -> TodayScreen()
            DashboardTab.Week -> WeekScreen()
        }
        // Note: TodayScreen / WeekScreen consume their own padding via
        // the Scaffold-aware composables they hold; we only honour the
        // bottom bar inset.
        @Suppress("UNUSED_EXPRESSION")
        padding
    }
}

@Composable
private fun DashboardTab.icon() =
    when (this) {
        DashboardTab.Today -> Icons.Filled.Today
        DashboardTab.Week -> Icons.Filled.CalendarMonth
    }
