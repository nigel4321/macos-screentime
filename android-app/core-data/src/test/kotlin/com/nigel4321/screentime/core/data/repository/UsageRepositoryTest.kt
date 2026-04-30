package com.nigel4321.screentime.core.data.repository

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.domain.model.BundleId
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import java.time.Clock
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

class UsageRepositoryTest {
    private lateinit var server: MockWebServer
    private lateinit var api: ScreentimeApi
    private lateinit var dao: FakeUsageSummaryDao
    private lateinit var clock: MutableClock

    private val from = Instant.parse("2026-04-22T00:00:00Z")
    private val to = Instant.parse("2026-04-29T00:00:00Z")

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
        dao = FakeUsageSummaryDao()
        clock = MutableClock(Instant.parse("2026-04-29T12:00:00Z"))
    }

    @After
    fun tearDown() {
        server.shutdown()
    }

    private fun repository() = UsageRepository(api, dao, clock)

    @Test
    fun `summary emits empty when cache is empty`() =
        runTest {
            val emission = repository().summary(from, to).first()

            assertEquals(0, emission.rows.size)
        }

    @Test
    fun `refresh stores rows under the cache key and summary emits them`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"results":[
                        {"bundle_id":"com.example.app","display_name":"Example","duration_seconds":3600},
                        {"day":"2026-04-29","duration_seconds":7200}
                    ]}
                    """.trimIndent(),
                ),
            )

            val repo = repository()
            repo.refresh(from, to, UsageRepository.GroupBy.BundleIdAndDay)
            val summary = repo.summary(from, to, UsageRepository.GroupBy.BundleIdAndDay).first()

            assertEquals(2, summary.rows.size)
            assertEquals(BundleId("com.example.app"), summary.rows[0].bundleId)
            assertEquals("Example", summary.rows[0].displayName)
            assertEquals(3600.seconds, summary.rows[0].duration)
            assertEquals(LocalDate.of(2026, 4, 29), summary.rows[1].day)
            // Day-grouped row has no display_name in the response — UI
            // falls back to bundle id, but bundle id is also absent so
            // null is correct.
            assertEquals(null, summary.rows[1].displayName)
        }

    @Test
    fun `refresh wipes prior rows for the same cache key before inserting new ones`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"results":[{"bundle_id":"old","duration_seconds":1}]}""",
                ),
            )
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"results":[{"bundle_id":"new","duration_seconds":2}]}""",
                ),
            )

            val repo = repository()
            repo.refresh(from, to)
            repo.refresh(from, to)
            val summary = repo.summary(from, to).first()

            assertEquals(1, summary.rows.size)
            assertEquals(BundleId("new"), summary.rows[0].bundleId)
        }

    @Test
    fun `isStale is true when no cached row exists`() =
        runTest {
            assertTrue(repository().isStale(from, to))
        }

    @Test
    fun `isStale is false within TTL after refresh`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody("""{"results":[]}"""),
            )
            val repo = repository()
            repo.refresh(from, to)

            clock.advance(2.minutes)

            assertFalse(repo.isStale(from, to))
        }

    @Test
    fun `isStale is true past TTL after refresh`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody("""{"results":[]}"""),
            )
            val repo = repository()
            repo.refresh(from, to)

            clock.advance(6.minutes)

            assertTrue(repo.isStale(from, to))
        }

    @Test
    fun `purgeOlderThan removes rows older than the cutoff and returns the deleted count`() =
        runTest {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"results":[{"bundle_id":"old","duration_seconds":1}]}""",
                ),
            )
            val repo = repository()
            repo.refresh(from, to)

            clock.advance(2.minutes)
            val cutoff = Instant.ofEpochMilli(clock.millis())

            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"results":[{"bundle_id":"new","duration_seconds":2}]}""",
                ),
            )
            // Bump time so the second refresh writes rows newer than the cutoff.
            clock.advance(1.minutes)
            // Refresh under a different cache key so the old rows aren't replaced.
            repo.refresh(from, to, UsageRepository.GroupBy.BundleId)

            val deleted = repo.purgeOlderThan(cutoff)

            assertEquals(1, deleted)
            // Surviving row is the freshly written one under the second cache key.
            val remaining = dao.snapshot()
            assertEquals(1, remaining.size)
            assertEquals("new", remaining[0].bundleId)
        }

    private class MutableClock(start: Instant) : Clock() {
        @Volatile private var now: Instant = start

        override fun getZone() = ZoneOffset.UTC

        override fun withZone(zone: java.time.ZoneId) = this

        override fun instant(): Instant = now

        fun advance(by: kotlin.time.Duration) {
            now = now.plusMillis(by.inWholeMilliseconds)
        }
    }
}
