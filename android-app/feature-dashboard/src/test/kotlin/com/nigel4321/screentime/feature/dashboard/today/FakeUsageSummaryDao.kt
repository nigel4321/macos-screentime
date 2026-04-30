package com.nigel4321.screentime.feature.dashboard.today

import com.nigel4321.screentime.core.data.cache.CacheMetadataEntity
import com.nigel4321.screentime.core.data.cache.UsageSummaryDao
import com.nigel4321.screentime.core.data.cache.UsageSummaryRowEntity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.map

internal class FakeUsageSummaryDao : UsageSummaryDao {
    private val rowState = MutableStateFlow<List<UsageSummaryRowEntity>>(emptyList())
    private val metadata = mutableMapOf<String, Long>()
    private var nextId = 1L

    override fun observeByCacheKey(cacheKey: String): Flow<List<UsageSummaryRowEntity>> =
        rowState.map { rows -> rows.filter { it.cacheKey == cacheKey } }

    override suspend fun lastRefreshAt(cacheKey: String): Long? = metadata[cacheKey]

    override suspend fun insertAll(rows: List<UsageSummaryRowEntity>) {
        rowState.value = rowState.value + rows.map { it.copy(id = nextId++) }
    }

    override suspend fun upsertMetadata(metadata: CacheMetadataEntity) {
        this.metadata[metadata.cacheKey] = metadata.lastRefreshAt
    }

    override suspend fun deleteByCacheKey(cacheKey: String) {
        rowState.value = rowState.value.filterNot { it.cacheKey == cacheKey }
    }

    override suspend fun deleteOlderThan(before: Long): Int {
        val toRemove = rowState.value.filter { it.cachedAt < before }
        rowState.value = rowState.value - toRemove.toSet()
        return toRemove.size
    }
}
