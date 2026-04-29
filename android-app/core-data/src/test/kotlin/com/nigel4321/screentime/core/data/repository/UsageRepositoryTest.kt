package com.nigel4321.screentime.core.data.repository

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.domain.model.BundleId
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import java.time.Instant
import java.time.LocalDate
import kotlin.time.Duration.Companion.seconds

class UsageRepositoryTest {
    private lateinit var server: MockWebServer
    private lateinit var repository: UsageRepository

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
        repository = UsageRepository(api)
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    @Test
    fun `summary maps bundle_id and day rows into domain types`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"results":[
                        {"bundle_id":"com.example.app","duration_seconds":3600},
                        {"day":"2026-04-29","duration_seconds":7200}
                    ]}
                    """.trimIndent(),
                ),
            )

            val result =
                repository.summary(
                    from = Instant.parse("2026-04-22T00:00:00Z"),
                    to = Instant.parse("2026-04-29T00:00:00Z"),
                    groupBy = UsageRepository.GroupBy.BundleIdAndDay,
                )

            assertEquals(2, result.rows.size)
            assertEquals(BundleId("com.example.app"), result.rows[0].bundleId)
            assertEquals(3600.seconds, result.rows[0].duration)
            assertEquals(LocalDate.of(2026, 4, 29), result.rows[1].day)
            assertEquals(7200.seconds, result.rows[1].duration)
        }
}
