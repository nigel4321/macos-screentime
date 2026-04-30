package com.nigel4321.screentime.feature.dashboard.week

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.repository.UsageRepository
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.core.domain.model.UsageSummary
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import java.time.Clock
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneId
import javax.inject.Inject
import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds

/**
 * Cache-first / network-refresh Week dashboard.
 *
 * Twin of `TodayViewModel`:
 *   - Two queries, both bucketed across the same 7-day window:
 *       1. `groupBy=day` for the chart's per-day totals
 *       2. `groupBy=bundle_id` for the top-apps tile list
 *   - Each emits its own cache Flow; the projection combines both.
 *   - `refresh()` triggers both cache-key refreshes concurrently and
 *     awaits both (so the loading→loaded transition fires once).
 */
@HiltViewModel
class WeekViewModel
    @Inject
    constructor(
        private val repository: UsageRepository,
        private val clock: Clock,
    ) : ViewModel() {
        private val refreshState = MutableStateFlow(InternalState())

        val uiState: StateFlow<WeekUiState> =
            combine(
                repository.summary(window.from, window.to, GROUP_DAY),
                repository.summary(window.from, window.to, GROUP_BUNDLE),
                refreshState,
            ) { byDay, byBundle, refresh ->
                project(byDay, byBundle, refresh)
            }.stateIn(
                scope = viewModelScope,
                // Eager so uiState.value is current right after refresh()
                // returns — same pattern as TodayViewModel; the projection
                // is cheap and viewModelScope cancels on logout.
                started = SharingStarted.Eagerly,
                initialValue = WeekUiState.Loading,
            )

        suspend fun refresh() {
            if (refreshState.value.isInFlight) return
            refreshState.update { it.copy(isInFlight = true, lastError = null) }
            runCatching {
                // Sequential rather than parallel keeps the OkHttp client's
                // single-host queue depth low and avoids interleaving two
                // simultaneous Retrofit calls during a busy refresh; both
                // round-trips are short.
                repository.refresh(window.from, window.to, GROUP_DAY)
                repository.refresh(window.from, window.to, GROUP_BUNDLE)
            }.onSuccess {
                refreshState.update { it.copy(isInFlight = false, lastError = null, hasFetched = true) }
            }.onFailure { error ->
                refreshState.update {
                    it.copy(
                        isInFlight = false,
                        lastError = error.localizedMessage ?: "Couldn't load this week's usage",
                    )
                }
            }
        }

        private fun project(
            byDay: UsageSummary,
            byBundle: UsageSummary,
            refresh: InternalState,
        ): WeekUiState {
            val total = byDay.rows.fold(Duration.ZERO) { acc, r -> acc + r.duration }

            return when {
                !refresh.hasFetched && refresh.lastError == null && byDay.rows.isEmpty() && byBundle.rows.isEmpty() ->
                    WeekUiState.Loading

                refresh.lastError != null && byDay.rows.isEmpty() && byBundle.rows.isEmpty() ->
                    WeekUiState.Error(refresh.lastError!!)

                byDay.rows.isEmpty() && byBundle.rows.isEmpty() -> WeekUiState.Empty

                else ->
                    WeekUiState.Loaded(
                        byDay = densifyByDay(byDay.rows),
                        topApps = byBundle.rows.sortedByDescending { it.duration }.take(MAX_TOP_APPS),
                        totalDuration = total.coerceAtLeast(0.seconds),
                        isRefreshing = refresh.isInFlight,
                    )
            }
        }

        /**
         * Fills in zero-duration buckets for any day in the window that the
         * backend didn't return — the chart is much easier to read with
         * a contiguous 7-bar series than gaps. Produces exactly 7 buckets,
         * sorted ascending by date.
         */
        private fun densifyByDay(rows: List<UsageRow>): List<DayBucket> {
            val byDate = rows.mapNotNull { row -> row.day?.let { it to row.duration } }.toMap()
            val zone = ZoneId.systemDefault()
            val today = LocalDate.now(clock.withZone(zone))
            return (DAYS_IN_WEEK - 1 downTo 0).map { offset ->
                val date = today.minusDays(offset.toLong())
                DayBucket(day = date, duration = byDate[date] ?: Duration.ZERO)
            }
        }

        /**
         * Today inclusive, looking back 7 days — same range the chart and
         * top-apps tiles render. `toString()`s on Instants are RFC-3339,
         * which matches the backend's `time.RFC3339` parse.
         */
        private val window: WeekWindow
            get() {
                val zone = ZoneId.systemDefault()
                val startOfRange =
                    LocalDate.now(clock.withZone(zone))
                        .minusDays((DAYS_IN_WEEK - 1).toLong())
                        .atStartOfDay(zone)
                        .toInstant()
                return WeekWindow(from = startOfRange, to = clock.instant())
            }

        private data class WeekWindow(val from: Instant, val to: Instant)

        private data class InternalState(
            val isInFlight: Boolean = false,
            val hasFetched: Boolean = false,
            val lastError: String? = null,
        )

        private fun MutableStateFlow<InternalState>.update(transform: (InternalState) -> InternalState) {
            value = transform(value)
        }

        private companion object {
            val GROUP_DAY = UsageRepository.GroupBy.Day
            val GROUP_BUNDLE = UsageRepository.GroupBy.BundleId
            const val DAYS_IN_WEEK = 7
            const val MAX_TOP_APPS = 4
        }
    }
