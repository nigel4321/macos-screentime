package com.nigel4321.screentime.feature.dashboard

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier

@Composable
fun TodayScreen() {
    Centered(text = "Today — implemented in §2.17")
}

@Composable
fun WeekScreen() {
    Centered(text = "Week — implemented in §2.18")
}

@Composable
private fun Centered(text: String) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        Text(text = text)
    }
}
