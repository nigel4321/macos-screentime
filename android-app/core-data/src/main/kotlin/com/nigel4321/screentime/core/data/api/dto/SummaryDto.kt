package com.nigel4321.screentime.core.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class SummaryResponse(
    @SerialName("results") val results: List<SummaryRowDto>,
)

@Serializable
data class SummaryRowDto(
    @SerialName("bundle_id") val bundleId: String? = null,
    @SerialName("day") val day: String? = null,
    @SerialName("duration_seconds") val durationSeconds: Long,
)
