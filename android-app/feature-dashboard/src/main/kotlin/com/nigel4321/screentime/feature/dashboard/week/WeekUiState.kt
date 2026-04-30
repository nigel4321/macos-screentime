package com.nigel4321.screentime.feature.dashboard.week

import com.nigel4321.screentime.core.domain.model.UsageRow
import java.time.LocalDate
import kotlin.time.Duration

/**
 * Snapshot of the Week dashboard. Mirrors [com.nigel4321.screentime.feature.dashboard.today.TodayUiState]
 * for state-machine symmetry; the Loaded shape differs because Week
 * surfaces both a per-day aggregate (for the chart) and a per-bundle
 * top-apps aggregate (for the small tiles).
 */
sealed interface WeekUiState {
    data object Loading : WeekUiState

    data object Empty : WeekUiState

    data class Loaded(
        /** Per-day duration, sorted ascending by date. Length is 7. */
        val byDay: List<DayBucket>,
        /** Top apps over the whole week, sorted desc by duration. */
        val topApps: List<UsageRow>,
        val totalDuration: Duration,
        val isRefreshing: Boolean = false,
    ) : WeekUiState

    data class Error(val message: String) : WeekUiState
}

data class DayBucket(
    val day: LocalDate,
    val duration: Duration,
)
