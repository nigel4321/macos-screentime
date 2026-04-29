package com.nigel4321.macosscreentime

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavGraphBuilder
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.nigel4321.macosscreentime.AuthGateViewModel.AuthGateState
import com.nigel4321.screentime.core.ui.theme.ScreentimeTheme
import com.nigel4321.screentime.feature.dashboard.TodayScreen
import com.nigel4321.screentime.feature.dashboard.WeekScreen
import com.nigel4321.screentime.feature.onboarding.OnboardingScreen
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
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
    const val TODAY = "today"
    const val WEEK = "week"
}

@Composable
private fun ScreentimeNavHost(
    modifier: Modifier = Modifier,
    authViewModel: AuthGateViewModel = androidx.hilt.navigation.compose.hiltViewModel(),
) {
    val navController = rememberNavController()
    val authState by authViewModel.authGate.collectAsStateWithLifecycle()

    val startDestination =
        when (authState) {
            AuthGateState.Authenticated -> Routes.TODAY
            AuthGateState.Anonymous -> Routes.ONBOARDING
        }

    NavHost(
        navController = navController,
        startDestination = startDestination,
        modifier = modifier,
    ) {
        screentimeGraph(navController)
    }

    // When the user signs out (Authenticated → Anonymous) or signs in
    // (Anonymous → Authenticated), route there instead of leaving the
    // user on a stale screen.
    androidx.compose.runtime.LaunchedEffect(authState) {
        val target =
            when (authState) {
                AuthGateState.Authenticated -> Routes.TODAY
                AuthGateState.Anonymous -> Routes.ONBOARDING
            }
        if (navController.currentDestination?.route != target) {
            navController.navigate(target) {
                popUpTo(navController.graph.id) { inclusive = true }
                launchSingleTop = true
            }
        }
    }
}

private fun NavGraphBuilder.screentimeGraph(navController: NavHostController) {
    composable(Routes.ONBOARDING) {
        OnboardingScreen(
            onAuthenticated = {
                navController.navigate(Routes.TODAY) {
                    popUpTo(Routes.ONBOARDING) { inclusive = true }
                    launchSingleTop = true
                }
            },
        )
    }
    composable(Routes.TODAY) { TodayScreen() }
    composable(Routes.WEEK) { WeekScreen() }
}
