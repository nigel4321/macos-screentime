package com.nigel4321.screentime.core.data.api

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.dto.GoogleAuthRequest
import com.nigel4321.screentime.core.data.api.dto.PairCompleteRequest
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit

class ScreentimeApiTest {
    private lateinit var server: MockWebServer
    private lateinit var api: ScreentimeApi

    @Before
    fun setUp() {
        server = MockWebServer().apply { start() }
        val json =
            Json {
                ignoreUnknownKeys = true
                explicitNulls = false
            }
        api =
            Retrofit.Builder()
                .baseUrl(server.url("/"))
                .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
                .build()
                .create(ScreentimeApi::class.java)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `authGoogle posts id_token and parses token response`() =
        runTest {
            server.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setBody("""{"token":"jwt-abc","expires_at":"2026-05-01T00:00:00Z"}"""),
            )

            val response = api.authGoogle(GoogleAuthRequest(idToken = "google-id-token"))

            assertEquals("jwt-abc", response.token)
            assertEquals("2026-05-01T00:00:00Z", response.expiresAt)

            val recorded = server.takeRequest()
            assertEquals("POST", recorded.method)
            assertEquals("/v1/auth/google", recorded.path)
            assertEquals("""{"id_token":"google-id-token"}""", recorded.body.readUtf8())
        }

    @Test
    fun `pairComplete posts code and parses token response`() =
        runTest {
            server.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setBody("""{"token":"jwt-paired","expires_at":"2026-05-01T00:00:00Z"}"""),
            )

            val response = api.pairComplete(PairCompleteRequest(code = "123456"))

            assertEquals("jwt-paired", response.token)
            val recorded = server.takeRequest()
            assertEquals("POST", recorded.method)
            assertEquals("/v1/account:pair-complete", recorded.path)
            assertEquals("""{"code":"123456"}""", recorded.body.readUtf8())
        }

    @Test
    fun `usageSummary parses bundle_id, display_name, day rows with from-to-groupBy params`() =
        runTest {
            server.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setBody(
                        """
                        {"results":[
                            {"bundle_id":"com.example.app","display_name":"Example","duration_seconds":3600},
                            {"day":"2026-04-29","duration_seconds":7200}
                        ]}
                        """.trimIndent(),
                    ),
            )

            val response =
                api.usageSummary(
                    from = "2026-04-22T00:00:00Z",
                    to = "2026-04-29T00:00:00Z",
                    groupBy = "bundle_id,day",
                )

            assertEquals(2, response.results.size)
            assertEquals("com.example.app", response.results[0].bundleId)
            assertEquals("Example", response.results[0].displayName)
            assertEquals(3600L, response.results[0].durationSeconds)
            assertEquals("2026-04-29", response.results[1].day)
            // omitempty on the server side means absent in JSON → null on the client.
            assertEquals(null, response.results[1].displayName)

            val recorded = server.takeRequest()
            assertEquals("GET", recorded.method)
            assertNotNull(recorded.requestUrl)
            val url = recorded.requestUrl!!
            assertEquals("/v1/usage:summary", url.encodedPath)
            assertEquals("2026-04-22T00:00:00Z", url.queryParameter("from"))
            assertEquals("2026-04-29T00:00:00Z", url.queryParameter("to"))
            assertEquals("bundle_id,day", url.queryParameter("groupBy"))
        }

    @Test
    fun `usageSummary omits groupBy when null`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody("""{"results":[]}"""),
            )

            api.usageSummary(from = "a", to = "b", groupBy = null)

            val recorded = server.takeRequest()
            assertEquals(null, recorded.requestUrl?.queryParameter("groupBy"))
        }

    @Test
    fun `currentPolicy parses the empty v0 stub from the backend`() =
        runTest {
            server.enqueue(
                MockResponse()
                    .setResponseCode(200)
                    .setBody(
                        """{"version":0,"app_limits":[],"downtime_windows":[],"block_list":[]}""",
                    ),
            )

            val response = api.currentPolicy()

            assertEquals(0L, response.version)
            assertEquals(0, response.appLimits.size)
            assertEquals(0, response.downtimeWindows.size)
            assertEquals(0, response.blockList.size)
        }
}
