package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.mapper.toDomain
import com.nigel4321.screentime.core.domain.model.Policy
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PolicyRepository
    @Inject
    constructor(
        private val api: ScreentimeApi,
    ) {
        suspend fun current(): Policy = api.currentPolicy().toDomain()
    }
