package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.auth.AuthState
import com.nigel4321.screentime.core.data.auth.TokenStore
import com.nigel4321.screentime.core.data.device.SelectedDeviceStore
import com.nigel4321.screentime.core.domain.model.DeviceId
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.distinctUntilChanged
import javax.inject.Inject
import javax.inject.Singleton

/**
 * The combined "where should the app land?" signal: anonymous, signed
 * in but no device chosen, or fully ready. The :app navigation gate
 * collects this to decide between onboarding / pairing / today.
 */
enum class SessionState {
    Anonymous,
    NeedsDevice,
    Ready,
}

@Singleton
class DeviceSession
    @Inject
    constructor(
        private val tokenStore: TokenStore,
        private val selectedDeviceStore: SelectedDeviceStore,
    ) {
        val state: Flow<SessionState> =
            combine(tokenStore.authState, selectedDeviceStore.selected) { auth, device ->
                when {
                    auth !is AuthState.Authenticated -> SessionState.Anonymous
                    device == null -> SessionState.NeedsDevice
                    else -> SessionState.Ready
                }
            }.distinctUntilChanged()

        fun current(): SessionState = compute(tokenStore, selectedDeviceStore.current())

        fun selectedDevice(): DeviceId? = selectedDeviceStore.current()

        private fun compute(
            tokenStore: TokenStore,
            device: DeviceId?,
        ): SessionState =
            when {
                tokenStore.authState.value !is AuthState.Authenticated -> SessionState.Anonymous
                device == null -> SessionState.NeedsDevice
                else -> SessionState.Ready
            }
    }
