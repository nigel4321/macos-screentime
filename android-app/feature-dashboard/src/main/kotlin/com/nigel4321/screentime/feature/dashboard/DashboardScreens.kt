package com.nigel4321.screentime.feature.dashboard

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.nigel4321.screentime.feature.dashboard.today.TodayScreen as RealTodayScreen

@Composable
fun TodayScreen() {
    RealTodayScreen()
}

@Composable
fun WeekScreen() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        Text(text = "Week — implemented in §2.18")
    }
}
