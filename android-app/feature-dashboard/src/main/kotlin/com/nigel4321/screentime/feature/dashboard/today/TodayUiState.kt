package com.nigel4321.screentime.feature.dashboard.today

import com.nigel4321.screentime.core.domain.model.UsageRow
import kotlin.time.Duration

/**
 * Snapshot of the Today dashboard. The single sealed type captures both
 * the lifecycle (loading / loaded / error / empty) and the auxiliary
 * `isRefreshing` flag so pull-to-refresh can re-render without flipping
 * the UI back to a full-screen spinner.
 */
sealed interface TodayUiState {
    data object Loading : TodayUiState

    data object Empty : TodayUiState

    data class Loaded(
        val rows: List<UsageRow>,
        val totalDuration: Duration,
        val isRefreshing: Boolean = false,
    ) : TodayUiState

    data class Error(val message: String) : TodayUiState
}
