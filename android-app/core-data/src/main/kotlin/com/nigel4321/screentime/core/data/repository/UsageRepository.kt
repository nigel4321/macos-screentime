package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.api.dto.SummaryRowDto
import com.nigel4321.screentime.core.data.cache.CacheKey
import com.nigel4321.screentime.core.data.cache.UsageSummaryDao
import com.nigel4321.screentime.core.data.cache.UsageSummaryRowEntity
import com.nigel4321.screentime.core.domain.model.BundleId
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.core.domain.model.UsageSummary
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import java.time.Clock
import java.time.Instant
import java.time.LocalDate
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

/**
 * Cache-first / network-refresh repository for `/v1/usage:summary`.
 *
 * - [summary] returns a cold [Flow] backed by the local Room cache. UI
 *   layers collect it and re-render whenever the cache mutates.
 * - [refresh] fetches a fresh summary from the backend and replaces the
 *   cached rows for the same [CacheKey] in a single transaction.
 * - [isStale] tells callers whether the cached value is older than [ttl]
 *   (default 5 minutes), so a ViewModel can decide whether to re-fetch
 *   on screen open or rely on cache.
 *
 * Cache invalidation rules:
 * - **Per-key replacement**: [refresh] always wipes-and-replaces rows for
 *   the matching [CacheKey], so a stale row never coexists with a fresh
 *   one for the same query.
 * - **TTL**: [isStale] reports `true` when the oldest `cached_at` for
 *   the key is older than [ttl]. A missing key is also "stale" so the
 *   first read triggers a fetch.
 * - **Global sweep**: [purgeOlderThan] lets callers drop everything
 *   older than a cutoff (used at app launch to prevent unbounded growth).
 */
@Singleton
class UsageRepository
    @Inject
    constructor(
        private val api: ScreentimeApi,
        private val dao: UsageSummaryDao,
        private val clock: Clock,
    ) {
        fun summary(
            from: Instant,
            to: Instant,
            groupBy: GroupBy = GroupBy.None,
        ): Flow<UsageSummary> =
            dao.observeByCacheKey(CacheKey.summary(from, to, groupBy)).map { rows ->
                UsageSummary(rows = rows.map { it.toDomain() })
            }

        suspend fun refresh(
            from: Instant,
            to: Instant,
            groupBy: GroupBy = GroupBy.None,
        ) {
            val response =
                api.usageSummary(
                    from = from.toString(),
                    to = to.toString(),
                    groupBy = groupBy.queryParam,
                )
            val cacheKey = CacheKey.summary(from, to, groupBy)
            val refreshedAt = clock.millis()
            val rows = response.results.map { it.toEntity(cacheKey, refreshedAt) }
            dao.replace(cacheKey, rows, refreshedAt)
        }

        suspend fun isStale(
            from: Instant,
            to: Instant,
            groupBy: GroupBy = GroupBy.None,
            ttl: Duration = DEFAULT_TTL,
        ): Boolean {
            val refreshedAt = dao.lastRefreshAt(CacheKey.summary(from, to, groupBy)) ?: return true
            val age = (clock.millis() - refreshedAt).coerceAtLeast(0L)
            return age >= ttl.inWholeMilliseconds
        }

        suspend fun purgeOlderThan(cutoff: Instant): Int = dao.deleteOlderThan(cutoff.toEpochMilli())

        enum class GroupBy(val queryParam: String?) {
            None(null),
            BundleId("bundle_id"),
            Day("day"),
            BundleIdAndDay("bundle_id,day"),
        }

        companion object {
            val DEFAULT_TTL: Duration = 5.minutes
        }
    }

private fun SummaryRowDto.toEntity(
    cacheKey: String,
    cachedAt: Long,
): UsageSummaryRowEntity =
    UsageSummaryRowEntity(
        cacheKey = cacheKey,
        bundleId = bundleId,
        day = day,
        displayName = displayName,
        durationSeconds = durationSeconds,
        cachedAt = cachedAt,
    )

internal fun UsageSummaryRowEntity.toDomain(): UsageRow =
    UsageRow(
        bundleId = bundleId?.let(::BundleId),
        day = day?.let(LocalDate::parse),
        duration = durationSeconds.seconds,
        displayName = displayName,
    )
