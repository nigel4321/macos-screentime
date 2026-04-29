package com.nigel4321.screentime.core.data.auth

import okhttp3.Interceptor
import okhttp3.Response
import javax.inject.Inject

/**
 * Adds `Authorization: Bearer <jwt>` to outgoing requests when a token
 * is available. Endpoints that don't require auth (e.g. `/v1/auth/google`)
 * are unaffected by header presence — the backend ignores it.
 */
class AuthInterceptor
    @Inject
    constructor(
        private val tokenStore: TokenStore,
    ) : Interceptor {
        override fun intercept(chain: Interceptor.Chain): Response {
            val token = tokenStore.current()
            val request =
                if (token != null) {
                    chain.request().newBuilder()
                        .header("Authorization", "Bearer $token")
                        .build()
                } else {
                    chain.request()
                }
            return chain.proceed(request)
        }
    }
