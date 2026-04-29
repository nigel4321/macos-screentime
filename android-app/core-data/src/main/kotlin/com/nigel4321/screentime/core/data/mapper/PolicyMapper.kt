package com.nigel4321.screentime.core.data.mapper

import com.nigel4321.screentime.core.data.api.dto.AppLimitDto
import com.nigel4321.screentime.core.data.api.dto.DowntimeWindowDto
import com.nigel4321.screentime.core.data.api.dto.PolicyResponse
import com.nigel4321.screentime.core.domain.model.AppLimit
import com.nigel4321.screentime.core.domain.model.BundleId
import com.nigel4321.screentime.core.domain.model.DowntimeWindow
import com.nigel4321.screentime.core.domain.model.Policy
import java.time.DayOfWeek
import java.time.LocalTime
import kotlin.time.Duration.Companion.seconds

internal fun PolicyResponse.toDomain(): Policy =
    Policy(
        version = version,
        appLimits = appLimits.map { it.toDomain() },
        downtimeWindows = downtimeWindows.map { it.toDomain() },
        blockList = blockList.map(::BundleId),
    )

private fun AppLimitDto.toDomain(): AppLimit =
    AppLimit(
        bundleId = BundleId(bundleId),
        dailyLimit = dailyLimitSeconds.seconds,
    )

private fun DowntimeWindowDto.toDomain(): DowntimeWindow =
    DowntimeWindow(
        start = LocalTime.parse(start),
        end = LocalTime.parse(end),
        days = days.map { DayOfWeek.valueOf(it.uppercase()) }.toSet(),
    )
