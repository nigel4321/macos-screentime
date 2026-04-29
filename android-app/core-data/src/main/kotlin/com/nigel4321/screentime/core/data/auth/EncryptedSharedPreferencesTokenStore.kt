package com.nigel4321.screentime.core.data.auth

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Persists the backend JWT in `EncryptedSharedPreferences` (AES-256-GCM
 * keys + values) and surfaces auth state as a [StateFlow] so the UI
 * layer can navigate on sign-in / sign-out without polling.
 *
 * Survives process death. The in-memory [StateFlow] is seeded from disk
 * at construction so the first observation reflects the persisted token.
 */
@Singleton
class EncryptedSharedPreferencesTokenStore
    @Inject
    constructor(
        @ApplicationContext context: Context,
    ) : TokenStore {
        private val prefs =
            run {
                val masterKey =
                    MasterKey.Builder(context)
                        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                        .build()
                EncryptedSharedPreferences.create(
                    context,
                    PREFS_NAME,
                    masterKey,
                    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
                )
            }

        private val state =
            MutableStateFlow<AuthState>(
                prefs.getString(KEY_TOKEN, null)
                    ?.let(AuthState::Authenticated)
                    ?: AuthState.Anonymous,
            )

        override val authState: StateFlow<AuthState> = state.asStateFlow()

        override fun current(): String? = (state.value as? AuthState.Authenticated)?.token

        override fun set(token: String?) {
            prefs.edit().apply {
                if (token != null) putString(KEY_TOKEN, token) else remove(KEY_TOKEN)
            }.apply()
            state.value = token?.let(AuthState::Authenticated) ?: AuthState.Anonymous
        }

        private companion object {
            const val PREFS_NAME = "screentime_auth"
            const val KEY_TOKEN = "backend_jwt"
        }
    }
