package com.nigel4321.macosscreentime

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
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
private fun ScreentimeNavHost(modifier: Modifier = Modifier) {
    val navController = rememberNavController()
    NavHost(
        navController = navController,
        startDestination = Routes.ONBOARDING,
        modifier = modifier,
    ) {
        composable(Routes.ONBOARDING) { OnboardingScreen() }
        composable(Routes.TODAY) { TodayScreen() }
        composable(Routes.WEEK) { WeekScreen() }
    }
}
