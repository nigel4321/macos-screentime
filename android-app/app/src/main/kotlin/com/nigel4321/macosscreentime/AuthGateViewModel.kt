package com.nigel4321.macosscreentime

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.auth.AuthState
import com.nigel4321.screentime.core.data.auth.TokenStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.stateIn
import javax.inject.Inject

@HiltViewModel
class AuthGateViewModel
    @Inject
    constructor(
        tokenStore: TokenStore,
    ) : ViewModel() {
        val authGate: StateFlow<AuthGateState> =
            tokenStore.authState
                .map { if (it is AuthState.Authenticated) AuthGateState.Authenticated else AuthGateState.Anonymous }
                .stateIn(
                    scope = viewModelScope,
                    started = SharingStarted.Eagerly,
                    initialValue =
                        if (tokenStore.authState.value is AuthState.Authenticated) {
                            AuthGateState.Authenticated
                        } else {
                            AuthGateState.Anonymous
                        },
                )

        enum class AuthGateState { Anonymous, Authenticated }
    }
