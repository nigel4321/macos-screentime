package com.nigel4321.screentime.core.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PolicyResponse(
    @SerialName("version") val version: Long,
    @SerialName("app_limits") val appLimits: List<AppLimitDto> = emptyList(),
    @SerialName("downtime_windows") val downtimeWindows: List<DowntimeWindowDto> = emptyList(),
    @SerialName("block_list") val blockList: List<String> = emptyList(),
)

@Serializable
data class AppLimitDto(
    @SerialName("bundle_id") val bundleId: String,
    @SerialName("daily_limit_seconds") val dailyLimitSeconds: Long,
)

@Serializable
data class DowntimeWindowDto(
    @SerialName("start") val start: String,
    @SerialName("end") val end: String,
    @SerialName("days") val days: List<String>,
)
