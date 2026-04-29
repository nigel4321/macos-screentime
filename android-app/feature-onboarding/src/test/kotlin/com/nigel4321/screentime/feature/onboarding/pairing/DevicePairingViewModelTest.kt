package com.nigel4321.screentime.feature.onboarding.pairing

import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.repository.DeviceRepository
import com.nigel4321.screentime.core.domain.model.DeviceId
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
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

@OptIn(ExperimentalCoroutinesApi::class)
class DevicePairingViewModelTest {
    private val dispatcher = StandardTestDispatcher()
    private lateinit var server: MockWebServer
    private lateinit var repository: DeviceRepository
    private lateinit var selected: InMemorySelectedDeviceStore

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
        repository = DeviceRepository(api)
        selected = InMemorySelectedDeviceStore()
    }

    @After
    fun tearDown() {
        server.shutdown()
        Dispatchers.resetMain()
    }

    private fun viewModel() = DevicePairingViewModel(repository, selected)

    @Test
    fun `init transitions Loading to Devices when backend returns devices`() =
        runTest(dispatcher) {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"devices":[
                        {"id":"d1","platform":"macos","fingerprint":"fp","created_at":"2026-04-28T12:00:00Z"}
                    ]}
                    """.trimIndent(),
                ),
            )

            val vm = viewModel()
            assertEquals(DevicePairingUiState.Loading, vm.uiState.value)

            advanceUntilIdle()

            val state = vm.uiState.value
            assertTrue("got $state", state is DevicePairingUiState.Devices)
            assertEquals(1, (state as DevicePairingUiState.Devices).devices.size)
        }

    @Test
    fun `init transitions Loading to ZeroDevices when backend returns empty list`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(200).setBody("""{"devices":[]}"""))

            val vm = viewModel()
            advanceUntilIdle()

            assertEquals(DevicePairingUiState.ZeroDevices, vm.uiState.value)
        }

    @Test
    fun `init transitions Loading to Error on network failure`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(500))

            val vm = viewModel()
            advanceUntilIdle()

            val state = vm.uiState.value
            assertTrue("got $state", state is DevicePairingUiState.Error)
        }

    @Test
    fun `selectAndContinue persists the chosen device id to the store`() =
        runTest(dispatcher) {
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"devices":[
                        {"id":"d1","platform":"macos","fingerprint":"fp","created_at":"2026-04-28T12:00:00Z"}
                    ]}
                    """.trimIndent(),
                ),
            )
            val vm = viewModel()
            advanceUntilIdle()

            vm.selectAndContinue(DeviceId("d1"))

            assertEquals(DeviceId("d1"), selected.current())
        }

    @Test
    fun `retry refetches and recovers from Error to Devices`() =
        runTest(dispatcher) {
            server.enqueue(MockResponse().setResponseCode(500))
            val vm = viewModel()
            advanceUntilIdle()
            assertTrue(vm.uiState.value is DevicePairingUiState.Error)

            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """
                    {"devices":[
                        {"id":"d1","platform":"macos","fingerprint":"fp","created_at":"2026-04-28T12:00:00Z"}
                    ]}
                    """.trimIndent(),
                ),
            )
            vm.retry()
            advanceUntilIdle()

            assertTrue(vm.uiState.value is DevicePairingUiState.Devices)
        }
}
