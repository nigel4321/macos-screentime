package com.nigel4321.screentime.core.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class DeviceListResponse(
    @SerialName("devices") val devices: List<DeviceDto>,
)

@Serializable
data class DeviceDto(
    @SerialName("id") val id: String,
    @SerialName("platform") val platform: String,
    @SerialName("fingerprint") val fingerprint: String,
    @SerialName("created_at") val createdAt: String,
    @SerialName("last_seen_at") val lastSeenAt: String? = null,
)
