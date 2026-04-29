package com.nigel4321.screentime.core.data.cache

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query
import androidx.room.Transaction
import kotlinx.coroutines.flow.Flow

@Dao
interface UsageSummaryDao {
    @Query("SELECT * FROM usage_summary_row WHERE cache_key = :cacheKey ORDER BY id")
    fun observeByCacheKey(cacheKey: String): Flow<List<UsageSummaryRowEntity>>

    @Query("SELECT last_refresh_at FROM cache_metadata WHERE cache_key = :cacheKey")
    suspend fun lastRefreshAt(cacheKey: String): Long?

    @Insert
    suspend fun insertAll(rows: List<UsageSummaryRowEntity>)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun upsertMetadata(metadata: CacheMetadataEntity)

    @Query("DELETE FROM usage_summary_row WHERE cache_key = :cacheKey")
    suspend fun deleteByCacheKey(cacheKey: String)

    @Query("DELETE FROM usage_summary_row WHERE cached_at < :before")
    suspend fun deleteOlderThan(before: Long): Int

    @Transaction
    suspend fun replace(
        cacheKey: String,
        rows: List<UsageSummaryRowEntity>,
        refreshedAt: Long,
    ) {
        deleteByCacheKey(cacheKey)
        insertAll(rows)
        upsertMetadata(CacheMetadataEntity(cacheKey = cacheKey, lastRefreshAt = refreshedAt))
    }
}
