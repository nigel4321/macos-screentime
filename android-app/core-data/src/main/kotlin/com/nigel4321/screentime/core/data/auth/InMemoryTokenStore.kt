package com.nigel4321.screentime.core.data.auth

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class InMemoryTokenStore
    @Inject
    constructor() : TokenStore {
        private val state = MutableStateFlow<AuthState>(AuthState.Anonymous)

        override val authState: StateFlow<AuthState> = state.asStateFlow()

        override fun current(): String? = (state.value as? AuthState.Authenticated)?.token

        override fun set(token: String?) {
            state.value = token?.let(AuthState::Authenticated) ?: AuthState.Anonymous
        }
    }
