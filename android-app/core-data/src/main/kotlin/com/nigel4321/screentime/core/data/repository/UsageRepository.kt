package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.mapper.toDomain
import com.nigel4321.screentime.core.domain.model.UsageSummary
import java.time.Instant
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class UsageRepository
    @Inject
    constructor(
        private val api: ScreentimeApi,
    ) {
        suspend fun summary(
            from: Instant,
            to: Instant,
            groupBy: GroupBy = GroupBy.None,
        ): UsageSummary =
            api.usageSummary(
                from = from.toString(),
                to = to.toString(),
                groupBy = groupBy.queryParam,
            ).toDomain()

        enum class GroupBy(val queryParam: String?) {
            None(null),
            BundleId("bundle_id"),
            Day("day"),
            BundleIdAndDay("bundle_id,day"),
        }
    }
