package com.nigel4321.screentime.core.data.cache

import com.nigel4321.screentime.core.data.repository.UsageRepository
import java.time.Instant

internal object CacheKey {
    fun summary(
        from: Instant,
        to: Instant,
        groupBy: UsageRepository.GroupBy,
    ): String = "summary|$from|$to|${groupBy.name}"
}
