package com.nigel4321.screentime.core.data.auth

import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test

class AuthAuthenticatorTest {
    private lateinit var server: MockWebServer

    @Before
    fun setUp() {
        server = MockWebServer().apply { start() }
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `clears the stored token and flips state to Anonymous on 401`() {
        val tokenStore = InMemoryTokenStore().apply { set("expired-jwt") }
        val client =
            OkHttpClient.Builder()
                .addInterceptor(AuthInterceptor(tokenStore))
                .authenticator(AuthAuthenticator(tokenStore))
                .build()

        // Enqueue 401 with no body. The authenticator returns null, so OkHttp
        // gives up and surfaces the 401 response to the caller.
        server.enqueue(MockResponse().setResponseCode(401))

        val response =
            client
                .newCall(Request.Builder().url(server.url("/v1/anything")).build())
                .execute()
        response.use {
            assertEquals(401, it.code)
        }

        assertNull(tokenStore.current())
        assertEquals(AuthState.Anonymous, tokenStore.authState.value)
    }
}
