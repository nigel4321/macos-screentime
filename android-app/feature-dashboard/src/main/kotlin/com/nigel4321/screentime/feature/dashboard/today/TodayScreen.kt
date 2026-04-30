package com.nigel4321.screentime.feature.dashboard.today

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.feature.dashboard.today.components.CategoriesTile
import com.nigel4321.screentime.feature.dashboard.today.components.DowntimeStatusTile
import com.nigel4321.screentime.feature.dashboard.today.components.EmptyState
import com.nigel4321.screentime.feature.dashboard.today.components.ErrorState
import com.nigel4321.screentime.feature.dashboard.today.components.LoadingSkeleton
import com.nigel4321.screentime.feature.dashboard.today.components.TopAppTile
import com.nigel4321.screentime.feature.dashboard.today.components.TotalUsageTile
import kotlin.time.Duration

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TodayScreen(
    modifier: Modifier = Modifier,
    viewModel: TodayViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()

    PullToRefreshBox(
        modifier = modifier.fillMaxSize(),
        isRefreshing = (state as? TodayUiState.Loaded)?.isRefreshing == true,
        onRefresh = viewModel::refresh,
    ) {
        when (val current = state) {
            TodayUiState.Loading -> LoadingSkeleton()
            TodayUiState.Empty -> EmptyState()
            is TodayUiState.Error -> ErrorState(message = current.message, onRetry = viewModel::refresh)
            is TodayUiState.Loaded -> LoadedBento(rows = current.rows, total = current.totalDuration)
        }
    }
}

@Composable
internal fun LoadedBento(
    rows: List<UsageRow>,
    total: Duration,
    modifier: Modifier = Modifier,
) {
    val topApps = rows.take(MAX_TOP_APPS)
    val maxDuration = topApps.firstOrNull()?.duration ?: Duration.ZERO

    LazyVerticalGrid(
        modifier = modifier.fillMaxSize(),
        columns = GridCells.Fixed(2),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item(span = { GridItemSpan(2) }) { TotalUsageTile(total = total) }

        // Top apps as 1×1 tiles. Two per row by default; if there's
        // an odd number, the last one occupies the row alone — which
        // is OK and avoids a half-empty cell. Cap at MAX_TOP_APPS so
        // the dashboard stays glanceable; the Week tab (§2.18) will
        // surface long-tail apps.
        itemsIndexed(topApps) { index, row ->
            TopAppTile(row = row, maxDuration = maxDuration, rank = index + 1)
        }

        item(span = { GridItemSpan(2) }) { CategoriesTile() }
        item(span = { GridItemSpan(2) }) { DowntimeStatusTile() }
    }
}

private const val MAX_TOP_APPS = 4
