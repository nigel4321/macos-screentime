package com.nigel4321.screentime.feature.dashboard.today

import kotlin.time.Duration

/**
 * Formats a [Duration] for the dashboard tiles. Choices:
 * - `< 1m`: `"<1m"` (anything under a minute is noise on a daily view)
 * - `< 1h`: `"42m"`
 * - `>= 1h`: `"3h 12m"` (drop minutes when zero: `"3h"`)
 */
internal fun Duration.formatHuman(): String {
    val totalMinutes = inWholeMinutes
    return when {
        totalMinutes < 1L -> "<1m"
        totalMinutes < HOUR_MINUTES -> "${totalMinutes}m"
        else -> {
            val hours = totalMinutes / HOUR_MINUTES
            val minutes = totalMinutes % HOUR_MINUTES
            if (minutes == 0L) "${hours}h" else "${hours}h ${minutes}m"
        }
    }
}

private const val HOUR_MINUTES = 60L
