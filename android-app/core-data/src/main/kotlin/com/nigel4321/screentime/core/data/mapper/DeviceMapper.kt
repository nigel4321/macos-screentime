package com.nigel4321.screentime.core.data.mapper

import com.nigel4321.screentime.core.data.api.dto.DeviceDto
import com.nigel4321.screentime.core.domain.model.Device
import com.nigel4321.screentime.core.domain.model.DeviceId
import com.nigel4321.screentime.core.domain.model.DevicePlatform
import java.time.Instant

internal fun DeviceDto.toDomain(): Device =
    Device(
        id = DeviceId(id),
        platform = parsePlatform(platform),
        fingerprint = fingerprint,
        createdAt = Instant.parse(createdAt),
        lastSeenAt = lastSeenAt?.let(Instant::parse),
    )

private fun parsePlatform(raw: String): DevicePlatform =
    when (raw) {
        "macos" -> DevicePlatform.Macos
        "android" -> DevicePlatform.Android
        else -> DevicePlatform.Unknown
    }
