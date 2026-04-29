package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.cache.UsageSummaryDao
import com.nigel4321.screentime.core.data.cache.UsageSummaryRowEntity
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.map

internal class FakeUsageSummaryDao : UsageSummaryDao {
    private val state = MutableStateFlow<List<UsageSummaryRowEntity>>(emptyList())
    private var nextId = 1L

    override fun observeByCacheKey(cacheKey: String): Flow<List<UsageSummaryRowEntity>> =
        state.map { rows -> rows.filter { it.cacheKey == cacheKey } }

    override suspend fun cachedAt(cacheKey: String): Long? {
        val matching = state.value.filter { it.cacheKey == cacheKey }
        return matching.minOfOrNull { it.cachedAt }
    }

    override suspend fun insertAll(rows: List<UsageSummaryRowEntity>) {
        state.value = state.value + rows.map { it.copy(id = nextId++) }
    }

    override suspend fun deleteByCacheKey(cacheKey: String) {
        state.value = state.value.filterNot { it.cacheKey == cacheKey }
    }

    override suspend fun deleteOlderThan(before: Long): Int {
        val toRemove = state.value.filter { it.cachedAt < before }
        state.value = state.value - toRemove.toSet()
        return toRemove.size
    }

    fun snapshot(): List<UsageSummaryRowEntity> = state.value
}
