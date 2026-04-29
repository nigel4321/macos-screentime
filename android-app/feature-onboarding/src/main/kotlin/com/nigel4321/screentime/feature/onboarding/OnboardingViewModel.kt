package com.nigel4321.screentime.feature.onboarding

import android.app.Activity
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.nigel4321.screentime.core.data.auth.AuthState
import com.nigel4321.screentime.core.data.auth.TokenStore
import com.nigel4321.screentime.core.data.repository.AuthRepository
import com.nigel4321.screentime.feature.onboarding.auth.GoogleSignInClient
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface OnboardingUiState {
    data object Idle : OnboardingUiState

    data object Loading : OnboardingUiState

    data class Error(val message: String) : OnboardingUiState

    data object Authenticated : OnboardingUiState
}

@HiltViewModel
class OnboardingViewModel
    @Inject
    constructor(
        private val googleSignInClient: GoogleSignInClient,
        private val authRepository: AuthRepository,
        tokenStore: TokenStore,
    ) : ViewModel() {
        private val transient = MutableStateFlow<OnboardingUiState>(OnboardingUiState.Idle)

        /**
         * Combines the persistent [TokenStore.authState] with the transient
         * sign-in state. If there's a stored token, the screen shows
         * [OnboardingUiState.Authenticated] so the host can navigate away,
         * regardless of whatever transient state is set.
         */
        val uiState: StateFlow<OnboardingUiState> =
            combine(tokenStore.authState, transient) { auth, transient ->
                if (auth is AuthState.Authenticated) OnboardingUiState.Authenticated else transient
            }.stateIn(
                scope = viewModelScope,
                started = SharingStarted.WhileSubscribed(STOP_TIMEOUT_MS),
                initialValue =
                    if (tokenStore.authState.value is AuthState.Authenticated) {
                        OnboardingUiState.Authenticated
                    } else {
                        OnboardingUiState.Idle
                    },
            )

        fun signIn(activity: Activity) {
            if (transient.value is OnboardingUiState.Loading) return
            transient.value = OnboardingUiState.Loading
            viewModelScope.launch {
                val tokenResult = googleSignInClient.requestSignIn(activity)
                tokenResult
                    .mapCatching { idToken -> authRepository.signInWithGoogle(idToken) }
                    .onSuccess {
                        // tokenStore.authState already drives the UI; reset
                        // transient back to Idle so it doesn't override.
                        transient.value = OnboardingUiState.Idle
                    }
                    .onFailure { error ->
                        transient.value =
                            OnboardingUiState.Error(
                                error.localizedMessage ?: "Sign-in failed",
                            )
                    }
            }
        }

        fun dismissError() {
            if (transient.value is OnboardingUiState.Error) {
                transient.value = OnboardingUiState.Idle
            }
        }

        /** Internal accessor used by the read-only view of [transient] for tests. */
        internal val transientState: StateFlow<OnboardingUiState> = transient.asStateFlow()

        private companion object {
            const val STOP_TIMEOUT_MS = 5_000L
        }
    }
