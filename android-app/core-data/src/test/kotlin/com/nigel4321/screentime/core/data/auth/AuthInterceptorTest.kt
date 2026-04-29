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

class AuthInterceptorTest {
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
    fun `adds Bearer header when a token is set`() {
        val tokenStore = InMemoryTokenStore().apply { set("jwt-abc") }
        val client = OkHttpClient.Builder().addInterceptor(AuthInterceptor(tokenStore)).build()
        server.enqueue(MockResponse().setResponseCode(200))

        client.newCall(Request.Builder().url(server.url("/v1/anything")).build()).execute().use {}

        val recorded = server.takeRequest()
        assertEquals("Bearer jwt-abc", recorded.getHeader("Authorization"))
    }

    @Test
    fun `does not add Bearer header when no token is set`() {
        val tokenStore = InMemoryTokenStore()
        val client = OkHttpClient.Builder().addInterceptor(AuthInterceptor(tokenStore)).build()
        server.enqueue(MockResponse().setResponseCode(200))

        client.newCall(Request.Builder().url(server.url("/v1/anything")).build()).execute().use {}

        val recorded = server.takeRequest()
        assertNull(recorded.getHeader("Authorization"))
    }
}
