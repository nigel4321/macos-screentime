package com.nigel4321.screentime.core.data.auth

import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import javax.inject.Inject

/**
 * Backend has no refresh-token endpoint by design — re-auth goes
 * through Google Sign-In + `/v1/auth/google` (or pair-complete) again.
 * On 401 we clear the stored token, flip [TokenStore.authState] to
 * [AuthState.Anonymous], and return null so OkHttp surfaces the 401
 * to callers; the UI layer observes [TokenStore.authState] and routes
 * to re-auth.
 */
class AuthAuthenticator
    @Inject
    constructor(
        private val tokenStore: TokenStore,
    ) : Authenticator {
        override fun authenticate(
            route: Route?,
            response: Response,
        ): Request? {
            tokenStore.set(null)
            return null
        }
    }
