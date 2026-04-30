package com.nigel4321.screentime.core.data.cache

import androidx.room.ColumnInfo
import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Cached row from a `/v1/usage:summary` response. The `cache_key`
 * column composites the (from, to, groupBy) request params so multiple
 * dashboard queries can coexist; `cached_at` is epoch millis used for
 * TTL invalidation.
 */
@Entity(
    tableName = "usage_summary_row",
    indices = [Index("cache_key")],
)
data class UsageSummaryRowEntity(
    @PrimaryKey(autoGenerate = true) val id: Long = 0L,
    @ColumnInfo("cache_key") val cacheKey: String,
    @ColumnInfo("bundle_id") val bundleId: String?,
    @ColumnInfo("day") val day: String?,
    @ColumnInfo("display_name") val displayName: String? = null,
    @ColumnInfo("duration_seconds") val durationSeconds: Long,
    @ColumnInfo("cached_at") val cachedAt: Long,
)
