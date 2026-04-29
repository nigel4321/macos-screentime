package com.nigel4321.macosscreentime

import android.app.Application
import dagger.hilt.android.HiltAndroidApp

/**
 * Application entry point. Annotated with @HiltAndroidApp so Hilt can
 * generate the component graph at compile time.
 */
@HiltAndroidApp
class ScreentimeApplication : Application()
