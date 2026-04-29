package com.nigel4321.screentime.core.domain.model

import java.time.DayOfWeek
import java.time.LocalTime
import kotlin.time.Duration

data class Policy(
    val version: Long,
    val appLimits: List<AppLimit>,
    val downtimeWindows: List<DowntimeWindow>,
    val blockList: List<BundleId>,
)

data class AppLimit(
    val bundleId: BundleId,
    val dailyLimit: Duration,
)

data class DowntimeWindow(
    val start: LocalTime,
    val end: LocalTime,
    val days: Set<DayOfWeek>,
)
