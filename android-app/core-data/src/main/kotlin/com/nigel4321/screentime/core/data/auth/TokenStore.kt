package com.nigel4321.screentime.core.data.auth

import kotlinx.coroutines.flow.StateFlow

/**
 * Holds the backend JWT and exposes auth state. Real persistent
 * implementation lands in §2.15 (EncryptedSharedPreferences); for §2.13
 * the in-memory implementation is enough to wire interceptors and
 * tests.
 */
interface TokenStore {
    val authState: StateFlow<AuthState>

    fun current(): String?

    fun set(token: String?)
}

sealed interface AuthState {
    data object Anonymous : AuthState

    data class Authenticated(val token: String) : AuthState
}
