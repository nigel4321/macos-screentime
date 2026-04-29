package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.mapper.toDomain
import com.nigel4321.screentime.core.domain.model.Device
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class DeviceRepository
    @Inject
    constructor(
        private val api: ScreentimeApi,
    ) {
        suspend fun list(): List<Device> = api.listDevices().devices.map { it.toDomain() }
    }
