package com.nigel4321.screentime.core.data.cache

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * Tracks when each cache_key was last refreshed. Lives separately from
 * `usage_summary_row` so an empty backend response (e.g. no usage on a
 * given day) still records freshness — without this, [UsageSummaryDao]
 * would report a refreshed-but-empty key as stale and the UI would
 * loop on background refreshes.
 */
@Entity(tableName = "cache_metadata")
data class CacheMetadataEntity(
    @PrimaryKey @ColumnInfo("cache_key") val cacheKey: String,
    @ColumnInfo("last_refresh_at") val lastRefreshAt: Long,
)
