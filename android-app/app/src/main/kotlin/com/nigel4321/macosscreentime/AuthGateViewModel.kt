package com.nigel4321.macosscreentime

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.repository.DeviceSession
import com.nigel4321.screentime.core.data.repository.SessionState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.stateIn
import javax.inject.Inject

@HiltViewModel
class AuthGateViewModel
    @Inject
    constructor(
        session: DeviceSession,
    ) : ViewModel() {
        val sessionState: StateFlow<SessionState> =
            session.state.stateIn(
                scope = viewModelScope,
                started = SharingStarted.Eagerly,
                initialValue = session.current(),
            )
    }
