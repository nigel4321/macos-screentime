package com.nigel4321.screentime.core.domain.model

import java.time.Instant

@JvmInline
value class DeviceId(val value: String)

enum class DevicePlatform { Macos, Android, Unknown }

data class Device(
    val id: DeviceId,
    val platform: DevicePlatform,
    val fingerprint: String,
    val createdAt: Instant,
    val lastSeenAt: Instant?,
)
