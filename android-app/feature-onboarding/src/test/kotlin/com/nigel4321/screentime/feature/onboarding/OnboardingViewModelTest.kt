package com.nigel4321.screentime.feature.onboarding

import android.app.Activity
import com.jakewharton.retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.nigel4321.screentime.core.data.api.ScreentimeApi
import com.nigel4321.screentime.core.data.auth.AuthState
import com.nigel4321.screentime.core.data.auth.InMemoryTokenStore
import com.nigel4321.screentime.core.data.repository.AuthRepository
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
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
import org.mockito.Mockito
import retrofit2.Retrofit

@OptIn(ExperimentalCoroutinesApi::class)
class OnboardingViewModelTest {
    private val dispatcher = StandardTestDispatcher()
    private lateinit var server: MockWebServer
    private lateinit var api: ScreentimeApi
    private lateinit var tokenStore: InMemoryTokenStore
    private lateinit var repository: AuthRepository
    private lateinit var google: FakeGoogleSignInClient
    private val activity: Activity = Mockito.mock(Activity::class.java)

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
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
        tokenStore = InMemoryTokenStore()
        repository = AuthRepository(api, tokenStore)
        google = FakeGoogleSignInClient()
    }

    @After
    fun tearDown() {
        server.shutdown()
        Dispatchers.resetMain()
    }

    private fun viewModel() = OnboardingViewModel(google, repository, tokenStore)

    @Test
    fun `initial state is Idle when token store is empty`() =
        runTest(dispatcher) {
            val vm = viewModel()
            assertEquals(OnboardingUiState.Idle, vm.uiState.first())
        }

    @Test
    fun `initial state is Authenticated when token store already has a token`() =
        runTest(dispatcher) {
            tokenStore.set("existing-jwt")

            val vm = viewModel()

            assertEquals(OnboardingUiState.Authenticated, vm.uiState.first())
        }

    @Test
    fun `signIn idle to loading to authenticated on success`() =
        runTest(dispatcher) {
            google.returns(Result.success("google-id-token"))
            server.enqueue(
                MockResponse().setResponseCode(200).setBody(
                    """{"token":"jwt-abc","expires_at":"2026-05-01T00:00:00Z"}""",
                ),
            )
            val vm = viewModel()

            vm.signIn(activity)
            assertEquals(OnboardingUiState.Loading, vm.transientState.value)

            advanceUntilIdle()

            assertEquals(AuthState.Authenticated("jwt-abc"), tokenStore.authState.value)
            assertEquals(OnboardingUiState.Authenticated, vm.uiState.first())
        }

    @Test
    fun `signIn surfaces Error when GoogleSignInClient fails`() =
        runTest(dispatcher) {
            google.returns(Result.failure(IllegalStateException("user cancelled")))
            val vm = viewModel()

            vm.signIn(activity)
            advanceUntilIdle()

            val state = vm.uiState.first()
            assertTrue("expected Error, got $state", state is OnboardingUiState.Error)
            assertEquals("user cancelled", (state as OnboardingUiState.Error).message)
        }

    @Test
    fun `signIn surfaces Error when backend rejects the id_token`() =
        runTest(dispatcher) {
            google.returns(Result.success("google-id-token"))
            server.enqueue(MockResponse().setResponseCode(401))
            val vm = viewModel()

            vm.signIn(activity)
            advanceUntilIdle()

            val state = vm.uiState.first()
            assertTrue(state is OnboardingUiState.Error)
            assertEquals(AuthState.Anonymous, tokenStore.authState.value)
        }

    @Test
    fun `dismissError returns to Idle`() =
        runTest(dispatcher) {
            google.returns(Result.failure(IllegalStateException("nope")))
            val vm = viewModel()

            vm.signIn(activity)
            advanceUntilIdle()
            assertTrue(vm.uiState.first() is OnboardingUiState.Error)

            vm.dismissError()
            advanceUntilIdle()

            assertEquals(OnboardingUiState.Idle, vm.uiState.first())
        }
}
