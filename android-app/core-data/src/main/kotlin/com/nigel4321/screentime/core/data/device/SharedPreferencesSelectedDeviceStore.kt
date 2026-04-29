package com.nigel4321.screentime.core.data.device

import android.content.Context
import com.nigel4321.screentime.core.domain.model.DeviceId
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class SharedPreferencesSelectedDeviceStore
    @Inject
    constructor(
        @ApplicationContext context: Context,
    ) : SelectedDeviceStore {
        private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

        private val state = MutableStateFlow<DeviceId?>(prefs.getString(KEY_DEVICE_ID, null)?.let(::DeviceId))

        override val selected: StateFlow<DeviceId?> = state.asStateFlow()

        override fun current(): DeviceId? = state.value

        override fun set(id: DeviceId?) {
            prefs.edit().apply {
                if (id != null) putString(KEY_DEVICE_ID, id.value) else remove(KEY_DEVICE_ID)
            }.apply()
            state.value = id
        }

        private companion object {
            const val PREFS_NAME = "screentime_device"
            const val KEY_DEVICE_ID = "selected_device_id"
        }
    }
