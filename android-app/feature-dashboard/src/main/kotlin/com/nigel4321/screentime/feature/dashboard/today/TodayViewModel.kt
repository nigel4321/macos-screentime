package com.nigel4321.screentime.feature.dashboard.today

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.repository.UsageRepository
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
 * Cache-first / network-refresh Today dashboard.
 *
 * - Observes the [UsageRepository] cache Flow keyed by today's
 *   `[startOfDay, now)` window grouped by `bundle_id` — the UI re-renders
 *   automatically when [UsageRepository.refresh] writes new rows.
 * - Refresh runs at construction. If the cache is stale per
 *   [UsageRepository.isStale]'s default 5-minute TTL we also fire a
 *   refresh on each [refresh] call (pull-to-refresh / retry).
 * - Errors land in [TodayUiState.Error] without clobbering whatever was
 *   in the cache before the failure: we keep the previous [TodayUiState.Loaded]
 *   if it exists and only flip to [TodayUiState.Error] when there's
 *   nothing to show.
 */
@HiltViewModel
class TodayViewModel
    @Inject
    constructor(
        private val repository: UsageRepository,
        private val clock: Clock,
    ) : ViewModel() {
        private val refreshState = MutableStateFlow(InternalState())

        val uiState: StateFlow<TodayUiState> =
            combine(repository.summary(today.from, today.to, GROUP), refreshState) { summary, refresh ->
                project(summary, refresh)
            }.stateIn(
                scope = viewModelScope,
                // Eager so `uiState.value` always reflects the latest
                // cache + refresh projection — including under
                // StandardTestDispatcher, where `WhileSubscribed`'s
                // lazy startup races with the test's `first()` call.
                // Cost is negligible: one screen, cheap projection,
                // viewModelScope cancels on logout.
                started = SharingStarted.Eagerly,
                initialValue = TodayUiState.Loading,
            )

        /**
         * Performs the refresh and suspends until the call completes.
         *
         * Public-suspend so tests can `await` it inside `runTest { … }`
         * without racing OkHttp's real-thread I/O — the previous
         * fire-and-forget shape had `advanceUntilIdle()` returning
         * before the HTTP response landed.
         *
         * Call sites:
         * - `TodayScreen` triggers initial load via `LaunchedEffect(Unit)`.
         * - Pull-to-refresh wraps it in `rememberCoroutineScope().launch`.
         */
        suspend fun refresh() {
            if (refreshState.value.isInFlight) return
            refreshState.update { it.copy(isInFlight = true, lastError = null) }
            runCatching { repository.refresh(today.from, today.to, GROUP) }
                .onSuccess {
                    refreshState.update { it.copy(isInFlight = false, lastError = null, hasFetched = true) }
                }
                .onFailure { error ->
                    refreshState.update {
                        it.copy(
                            isInFlight = false,
                            lastError = error.localizedMessage ?: "Couldn't load today's usage",
                        )
                    }
                }
        }

        private fun project(
            summary: UsageSummary,
            refresh: InternalState,
        ): TodayUiState {
            val total = summary.rows.fold(Duration.ZERO) { acc, r -> acc + r.duration }
            val sorted = summary.rows.sortedByDescending { it.duration }
            return when {
                // First load hasn't completed yet — keep showing skeleton until
                // either rows arrive or the fetch errors. This avoids the empty
                // state flashing while the network call is in flight.
                !refresh.hasFetched && refresh.lastError == null && summary.rows.isEmpty() ->
                    TodayUiState.Loading

                // Network failure with nothing cached — surface error.
                refresh.lastError != null && summary.rows.isEmpty() ->
                    TodayUiState.Error(refresh.lastError!!)

                summary.rows.isEmpty() -> TodayUiState.Empty

                else ->
                    TodayUiState.Loaded(
                        rows = sorted,
                        totalDuration = total.coerceAtLeast(0.seconds),
                        isRefreshing = refresh.isInFlight,
                    )
            }
        }

        private val today: TodayWindow
            get() {
                val zone = ZoneId.systemDefault()
                val startOfDay = LocalDate.now(clock.withZone(zone)).atStartOfDay(zone).toInstant()
                return TodayWindow(from = startOfDay, to = clock.instant())
            }

        private data class TodayWindow(val from: Instant, val to: Instant)

        /**
         * Mutable refresh-flight bookkeeping that the UI Flow combines with
         * the cache Flow. Kept private so callers can't reach in to alter
         * `hasFetched`, which guards the loading-vs-empty distinction.
         */
        private data class InternalState(
            val isInFlight: Boolean = false,
            val hasFetched: Boolean = false,
            val lastError: String? = null,
        )

        private fun MutableStateFlow<InternalState>.update(transform: (InternalState) -> InternalState) {
            value = transform(value)
        }

        private companion object {
            val GROUP = UsageRepository.GroupBy.BundleId
        }
    }
