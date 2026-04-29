package com.nigel4321.screentime.core.data.repository

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.domain.model.DeviceId
import com.nigel4321.screentime.core.domain.model.DevicePlatform
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import java.time.Instant

class DeviceRepositoryTest {
    private lateinit var server: MockWebServer
    private lateinit var repository: DeviceRepository

    @Before
    fun setUp() {
        server = MockWebServer().apply { start() }
        val json =
            Json {
                ignoreUnknownKeys = true
                explicitNulls = false
            }
        val api =
            Retrofit.Builder()
                .baseUrl(server.url("/"))
                .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
                .build()
                .create(ScreentimeApi::class.java)
        repository = DeviceRepository(api)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `list maps devices into domain types`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"devices":[
                        {"id":"d1","platform":"macos","fingerprint":"fp-mac","created_at":"2026-04-28T12:00:00Z","last_seen_at":"2026-04-29T09:00:00Z"},
                        {"id":"d2","platform":"android","fingerprint":"fp-droid","created_at":"2026-04-28T12:00:00Z"}
                    ]}
                    """.trimIndent(),
                ),
            )

            val devices = repository.list()

            assertEquals(2, devices.size)
            assertEquals(DeviceId("d1"), devices[0].id)
            assertEquals(DevicePlatform.Macos, devices[0].platform)
            assertEquals(Instant.parse("2026-04-28T12:00:00Z"), devices[0].createdAt)
            assertEquals(Instant.parse("2026-04-29T09:00:00Z"), devices[0].lastSeenAt)
            assertEquals(DevicePlatform.Android, devices[1].platform)
            assertNull(devices[1].lastSeenAt)
        }

    @Test
    fun `list returns empty list when backend returns empty array`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody("""{"devices":[]}"""),
            )

            assertEquals(0, repository.list().size)
        }

    @Test
    fun `list maps unknown platform to DevicePlatform Unknown`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"devices":[
                        {"id":"d1","platform":"linux","fingerprint":"fp","created_at":"2026-04-28T12:00:00Z"}
                    ]}
                    """.trimIndent(),
                ),
            )

            val devices = repository.list()

            assertEquals(DevicePlatform.Unknown, devices[0].platform)
        }
}
