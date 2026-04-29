package com.nigel4321.screentime.core.data.repository

import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.api.dto.GoogleAuthRequest
import com.nigel4321.screentime.core.data.api.dto.PairCompleteRequest
import com.nigel4321.screentime.core.data.auth.TokenStore
import com.nigel4321.screentime.core.domain.model.BackendJwt
import com.nigel4321.screentime.core.domain.model.PairingCode
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepository
    @Inject
    constructor(
        private val api: ScreentimeApi,
        private val tokenStore: TokenStore,
    ) {
        suspend fun signInWithGoogle(idToken: String): BackendJwt {
            val response = api.authGoogle(GoogleAuthRequest(idToken = idToken))
            tokenStore.set(response.token)
            return BackendJwt(response.token)
        }

        suspend fun completePairing(code: PairingCode): BackendJwt {
            val response = api.pairComplete(PairCompleteRequest(code = code.code))
            tokenStore.set(response.token)
            return BackendJwt(response.token)
        }

        fun signOut() {
            tokenStore.set(null)
        }
    }
