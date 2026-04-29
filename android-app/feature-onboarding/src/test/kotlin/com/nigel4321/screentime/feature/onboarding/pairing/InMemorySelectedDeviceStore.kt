package com.nigel4321.screentime.feature.onboarding.pairing

import com.nigel4321.screentime.core.data.device.SelectedDeviceStore
import com.nigel4321.screentime.core.domain.model.DeviceId
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

internal class InMemorySelectedDeviceStore : SelectedDeviceStore {
    private val state = MutableStateFlow<DeviceId?>(null)
    override val selected: StateFlow<DeviceId?> = state.asStateFlow()

    override fun current(): DeviceId? = state.value

    override fun set(id: DeviceId?) {
        state.value = id
    }
}
