package com.nigel4321.screentime.feature.dashboard.week

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.repository.UsageRepository
import com.nigel4321.screentime.feature.dashboard.today.FakeUsageSummaryDao
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.launch
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import retrofit2.Retrofit
import java.time.Clock
import java.time.Instant
import java.time.ZoneOffset
import kotlin.time.Duration.Companion.minutes

@OptIn(ExperimentalCoroutinesApi::class)
class WeekViewModelTest {
    private val dispatcher = StandardTestDispatcher()
    private lateinit var server: MockWebServer
    private lateinit var repository: UsageRepository
    private lateinit var clock: Clock

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
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
        clock = Clock.fixed(Instant.parse("2026-04-30T12:00:00Z"), ZoneOffset.UTC)
        repository = UsageRepository(api, FakeUsageSummaryDao(), clock)
    }

    @After
    fun tearDown() {
        server.shutdown()
        Dispatchers.resetMain()
    }

    private fun viewModel() = WeekViewModel(repository, clock)

    /**
     * Each refresh fires two requests (groupBy=day + groupBy=bundle_id),
     * so tests need to enqueue two responses per refresh().
     */
    private fun enqueueOk(body: String = """{"results":[]}""") {
        server.enqueue(MockResponse().setResponseCode(200).setBody(body))
    }

    @Test
    fun `initial state is Loading before any refresh runs`() =
        runTest(dispatcher) {
            val vm = viewModel()
            assertEquals(WeekUiState.Loading, vm.uiState.value)
        }

    @Test
    fun `refresh transitions Loading to Empty when both queries return no rows`() =
        runTest(dispatcher) {
            enqueueOk()
            enqueueOk()
            val vm = viewModel()

            vm.refresh()
            advanceUntilIdle()

            assertEquals(WeekUiState.Empty, vm.uiState.value)
        }

    @Test
    fun `refresh transitions Loading to Loaded with densified 7 days and top apps`() =
        runTest(dispatcher) {
            // Day-grouped response: only two days have data; the
            // remaining 5 must be filled in with zero-duration buckets.
            enqueueOk(
                """
                {"results":[
                    {"day":"2026-04-29","duration_seconds":3600},
                    {"day":"2026-04-30","duration_seconds":1800}
                ]}
                """.trimIndent(),
            )
            enqueueOk(
                """
                {"results":[
                    {"bundle_id":"com.a","duration_seconds":4200},
                    {"bundle_id":"com.b","duration_seconds":1200}
                ]}
                """.trimIndent(),
            )
            val vm = viewModel()

            vm.refresh()
            advanceUntilIdle()

            val state = vm.uiState.value
            assertTrue("got $state", state is WeekUiState.Loaded)
            val loaded = state as WeekUiState.Loaded
            assertEquals(7, loaded.byDay.size)
            // Last bucket is today (clock fixed at 2026-04-30).
            assertEquals("2026-04-30", loaded.byDay.last().day.toString())
            // Day order is ascending — earliest first.
            assertTrue(loaded.byDay.first().day.isBefore(loaded.byDay.last().day))
            // Top apps sorted desc by duration.
            assertEquals("com.a", loaded.topApps[0].bundleId?.value)
            // Total = 3600 + 1800 = 5400s = 90m.
            assertEquals(90.minutes, loaded.totalDuration)
        }

    @Test
    fun `refresh transitions Loading to Error when the first request fails`() =
        runTest(dispatcher) {
            // First (groupBy=day) fails; second is never enqueued.
            server.enqueue(MockResponse().setResponseCode(500))
            val vm = viewModel()

            vm.refresh()
            advanceUntilIdle()

            assertTrue(vm.uiState.value is WeekUiState.Error)
        }

    @Test
    fun `refresh recovers from Error to Loaded`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(500))
            val vm = viewModel()
            vm.refresh()
            advanceUntilIdle()
            assertTrue(vm.uiState.value is WeekUiState.Error)

            // Second attempt succeeds — both queries have to land.
            enqueueOk("""{"results":[{"day":"2026-04-30","duration_seconds":600}]}""")
            enqueueOk("""{"results":[{"bundle_id":"com.a","duration_seconds":600}]}""")
            vm.refresh()
            advanceUntilIdle()

            assertTrue(vm.uiState.value is WeekUiState.Loaded)
        }

    @Test
    fun `concurrent refresh while one is in-flight is a no-op`() =
        runTest(dispatcher) {
            // Only two responses queued (one full refresh worth) — a
            // second concurrent refresh would deadlock if not suppressed.
            enqueueOk()
            enqueueOk()
            val vm = viewModel()

            val first = launch { vm.refresh() }
            advanceUntilIdle()
            vm.refresh()
            first.join()
            advanceUntilIdle()

            // Two requests = exactly one full refresh, not two.
            assertEquals(2, server.requestCount)
        }
}
