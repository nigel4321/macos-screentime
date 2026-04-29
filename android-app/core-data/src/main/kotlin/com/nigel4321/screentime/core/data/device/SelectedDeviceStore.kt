package com.nigel4321.screentime.core.data.device

import com.nigel4321.screentime.core.domain.model.DeviceId
import kotlinx.coroutines.flow.StateFlow

/**
 * Persists the user's choice of "primary" device and surfaces it as a
 * [StateFlow] so the auth gate can route on changes. The selection is
 * not sensitive (the device id is server-issued and tied to the
 * authenticated account), so plain SharedPreferences is fine.
 */
interface SelectedDeviceStore {
    val selected: StateFlow<DeviceId?>

    fun current(): DeviceId?

    fun set(id: DeviceId?)
}
