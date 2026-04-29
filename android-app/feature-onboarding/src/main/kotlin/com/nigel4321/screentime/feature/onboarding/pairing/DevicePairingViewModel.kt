package com.nigel4321.screentime.feature.onboarding.pairing

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.device.SelectedDeviceStore
import com.nigel4321.screentime.core.data.repository.DeviceRepository
import com.nigel4321.screentime.core.domain.model.Device
import com.nigel4321.screentime.core.domain.model.DeviceId
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface DevicePairingUiState {
    data object Loading : DevicePairingUiState

    data class Devices(val devices: List<Device>) : DevicePairingUiState

    data object ZeroDevices : DevicePairingUiState

    data class Error(val message: String) : DevicePairingUiState
}

@HiltViewModel
class DevicePairingViewModel
    @Inject
    constructor(
        private val deviceRepository: DeviceRepository,
        private val selectedDeviceStore: SelectedDeviceStore,
    ) : ViewModel() {
        private val state = MutableStateFlow<DevicePairingUiState>(DevicePairingUiState.Loading)
        val uiState: StateFlow<DevicePairingUiState> = state.asStateFlow()

        init {
            load()
        }

        fun retry() {
            load()
        }

        fun selectAndContinue(id: DeviceId) {
            selectedDeviceStore.set(id)
            // The auth gate observes [SelectedDeviceStore.selected] and will
            // navigate away from the pairing screen automatically.
        }

        private fun load() {
            state.value = DevicePairingUiState.Loading
            viewModelScope.launch {
                runCatching { deviceRepository.list() }
                    .onSuccess { devices ->
                        state.value =
                            if (devices.isEmpty()) {
                                DevicePairingUiState.ZeroDevices
                            } else {
                                DevicePairingUiState.Devices(devices)
                            }
                    }
                    .onFailure { error ->
                        state.value =
                            DevicePairingUiState.Error(
                                error.localizedMessage ?: "Couldn't load devices",
                            )
                    }
            }
        }
    }
