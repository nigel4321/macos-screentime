package com.nigel4321.screentime.core.domain.model

import java.time.LocalDate
import kotlin.time.Duration

data class UsageSummary(
    val rows: List<UsageRow>,
)

data class UsageRow(
    val bundleId: BundleId?,
    val day: LocalDate?,
    val duration: Duration,
)
