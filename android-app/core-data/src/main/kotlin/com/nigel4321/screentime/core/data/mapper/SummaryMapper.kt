package com.nigel4321.screentime.core.data.mapper

import com.nigel4321.screentime.core.data.api.dto.SummaryResponse
import com.nigel4321.screentime.core.data.api.dto.SummaryRowDto
import com.nigel4321.screentime.core.domain.model.BundleId
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.core.domain.model.UsageSummary
import java.time.LocalDate
import kotlin.time.Duration.Companion.seconds

internal fun SummaryResponse.toDomain(): UsageSummary = UsageSummary(rows = results.map { it.toDomain() })

private fun SummaryRowDto.toDomain(): UsageRow =
    UsageRow(
        bundleId = bundleId?.let(::BundleId),
        day = day?.let(LocalDate::parse),
        duration = durationSeconds.seconds,
    )
