package com.nigel4321.screentime.feature.dashboard.week

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
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.rememberCoroutineScope
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
import com.nigel4321.screentime.feature.dashboard.week.components.TotalWeekTile
import com.nigel4321.screentime.feature.dashboard.week.components.WeekChartTile
import kotlinx.coroutines.launch
import kotlin.time.Duration

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WeekScreen(
    modifier: Modifier = Modifier,
    viewModel: WeekViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    val triggerRefresh: () -> Unit = { scope.launch { viewModel.refresh() } }

    LaunchedEffect(Unit) { viewModel.refresh() }

    PullToRefreshBox(
        modifier = modifier.fillMaxSize(),
        isRefreshing = (state as? WeekUiState.Loaded)?.isRefreshing == true,
        onRefresh = triggerRefresh,
    ) {
        when (val current = state) {
            WeekUiState.Loading -> LoadingSkeleton()
            WeekUiState.Empty -> EmptyState()
            is WeekUiState.Error -> ErrorState(message = current.message, onRetry = triggerRefresh)
            is WeekUiState.Loaded ->
                LoadedBento(
                    byDay = current.byDay,
                    topApps = current.topApps,
                    total = current.totalDuration,
                )
        }
    }
}

@Composable
internal fun LoadedBento(
    byDay: List<DayBucket>,
    topApps: List<UsageRow>,
    total: Duration,
    modifier: Modifier = Modifier,
) {
    val maxDuration = topApps.firstOrNull()?.duration ?: Duration.ZERO

    LazyVerticalGrid(
        modifier = modifier.fillMaxSize(),
        columns = GridCells.Fixed(2),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        item(span = { GridItemSpan(2) }) { TotalWeekTile(total = total) }
        item(span = { GridItemSpan(2) }) { WeekChartTile(days = byDay) }

        // Same 1×1 top-app tiles Today uses; reusing keeps the visual
        // language consistent across both tabs and means the §2.22
        // displayName fallback is exercised here too.
        itemsIndexed(topApps) { index, row ->
            TopAppTile(row = row, maxDuration = maxDuration, rank = index + 1)
        }

        item(span = { GridItemSpan(2) }) { CategoriesTile() }
        item(span = { GridItemSpan(2) }) { DowntimeStatusTile() }
    }
}
