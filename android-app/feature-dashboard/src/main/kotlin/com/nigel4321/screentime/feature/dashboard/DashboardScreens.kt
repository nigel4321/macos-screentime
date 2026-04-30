package com.nigel4321.screentime.feature.dashboard

import androidx.compose.runtime.Composable
import com.nigel4321.screentime.feature.dashboard.today.TodayScreen as RealTodayScreen
import com.nigel4321.screentime.feature.dashboard.week.WeekScreen as RealWeekScreen

@Composable
fun TodayScreen() {
    RealTodayScreen()
}

@Composable
fun WeekScreen() {
    RealWeekScreen()
}
