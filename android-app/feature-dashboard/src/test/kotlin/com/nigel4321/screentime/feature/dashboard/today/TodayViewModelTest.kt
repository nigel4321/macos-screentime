package com.nigel4321.screentime.feature.dashboard.today

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.repository.UsageRepository
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
class TodayViewModelTest {
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

    private fun viewModel() = TodayViewModel(repository, clock)

    @Test
    fun `initial state is Loading before any refresh runs`() =
        runTest(dispatcher) {
            val vm = viewModel()

            assertEquals(TodayUiState.Loading, vm.uiState.value)
        }

    @Test
    fun `refresh transitions Loading to Empty when backend returns no rows`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(200).setBody("""{"results":[]}"""))
            val vm = viewModel()

            vm.refresh()
            // Drain stateIn's upstream-collector dispatch so the combine
            // re-emission lands in vm.uiState.value before we read it.
            advanceUntilIdle()

            assertEquals(TodayUiState.Empty, vm.uiState.value)
        }

    @Test
    fun `refresh transitions Loading to Loaded when backend returns rows`() =
        runTest(dispatcher) {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"results":[
                        {"bundle_id":"com.a","duration_seconds":600},
                        {"bundle_id":"com.b","duration_seconds":1800}
                    ]}
                    """.trimIndent(),
                ),
            )
            val vm = viewModel()

            vm.refresh()
            advanceUntilIdle()

            val state = vm.uiState.value
            assertTrue("got $state", state is TodayUiState.Loaded)
            val loaded = state as TodayUiState.Loaded
            assertEquals(2, loaded.rows.size)
            // Sorted descending by duration: com.b (30m) before com.a (10m).
            assertEquals("com.b", loaded.rows[0].bundleId?.value)
            assertEquals(40.minutes, loaded.totalDuration)
        }

    @Test
    fun `refresh transitions Loading to Error when backend fails on first fetch`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(500))
            val vm = viewModel()

            vm.refresh()
            advanceUntilIdle()

            assertTrue(vm.uiState.value is TodayUiState.Error)
        }

    @Test
    fun `refresh recovers from Error to Loaded`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(500))
            val vm = viewModel()
            vm.refresh()
            advanceUntilIdle()
            assertTrue(vm.uiState.value is TodayUiState.Error)

            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"results":[{"bundle_id":"com.a","duration_seconds":600}]}""",
                ),
            )
            vm.refresh()
            advanceUntilIdle()

            assertTrue(vm.uiState.value is TodayUiState.Loaded)
        }

    @Test
    fun `concurrent refresh while one is in-flight is a no-op (does not double-fetch)`() =
        runTest(dispatcher) {
            // Only one response queued — a duplicate dispatch would hang.
            server.enqueue(
                MockResponse().setResponseCode(200).setBody("""{"results":[]}"""),
            )
            val vm = viewModel()

            // Launch a refresh on a child coroutine so we can fire a
            // second one while the first is still suspended on HTTP.
            // The InternalState `isInFlight` guard suppresses the second.
            val first = launch { vm.refresh() }
            advanceUntilIdle() // let the first call see isInFlight=true
            vm.refresh()
            first.join()
            advanceUntilIdle()

            assertEquals(1, server.requestCount)
            assertEquals(TodayUiState.Empty, vm.uiState.value)
        }

    @Test
    fun `refresh windows today using clock and system zone`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(200).setBody("""{"results":[]}"""))
            val vm = viewModel()

            vm.refresh()

            val recorded = server.takeRequest()
            val url = recorded.requestUrl ?: error("missing URL")
            val from = url.queryParameter("from")
            val to = url.queryParameter("to")
            // 'from' is start-of-day in the system zone; 'to' is now (UTC clock fixed).
            assertTrue("from looks wrong: $from", from?.startsWith("2026-04-") == true)
            assertEquals("2026-04-30T12:00:00Z", to)
            // Should default to grouping by bundle_id.
            assertEquals("bundle_id", url.queryParameter("groupBy"))
        }
}
