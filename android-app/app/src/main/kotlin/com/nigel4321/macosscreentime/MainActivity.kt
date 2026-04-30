package com.nigel4321.macosscreentime

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraphBuilder
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.nigel4321.screentime.core.data.repository.SessionState
import com.nigel4321.screentime.core.ui.theme.ScreentimeTheme
import com.nigel4321.screentime.feature.onboarding.OnboardingScreen
import com.nigel4321.screentime.feature.onboarding.pairing.DevicePairingScreen
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            ScreentimeTheme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    ScreentimeNavHost(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }
}

private object Routes {
    const val ONBOARDING = "onboarding"
    const val PAIRING = "pairing"

    // Single tabbed shell hosting both Today and Week — see DashboardHost.
    const val DASHBOARD = "dashboard"
}

@Composable
private fun ScreentimeNavHost(
    modifier: Modifier = Modifier,
    authViewModel: AuthGateViewModel = hiltViewModel(),
) {
    val navController = rememberNavController()
    val session by authViewModel.sessionState.collectAsStateWithLifecycle()

    NavHost(
        navController = navController,
        startDestination = session.toRoute(),
        modifier = modifier,
    ) {
        screentimeGraph()
    }

    // Re-route on subsequent state changes (sign-in, device-pick,
    // sign-out). popUpTo(graph) clears the back stack so users can't
    // swipe back into the previous flow.
    LaunchedEffect(session) {
        val target = session.toRoute()
        if (navController.currentDestination?.route != target) {
            navController.navigate(target) {
                popUpTo(navController.graph.id) { inclusive = true }
                launchSingleTop = true
            }
        }
    }
}

private fun NavGraphBuilder.screentimeGraph() {
    composable(Routes.ONBOARDING) { OnboardingScreen(onAuthenticated = {}) }
    composable(Routes.PAIRING) { DevicePairingScreen() }
    composable(Routes.DASHBOARD) { DashboardHost() }
}

private fun SessionState.toRoute(): String =
    when (this) {
        SessionState.Anonymous -> Routes.ONBOARDING
        SessionState.NeedsDevice -> Routes.PAIRING
        SessionState.Ready -> Routes.DASHBOARD
    }
