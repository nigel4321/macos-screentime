package com.nigel4321.screentime.feature.dashboard.today

import org.junit.Assert.assertEquals
import org.junit.Test
import kotlin.time.Duration.Companion.hours
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds

class FormatTest {
    @Test
    fun `under one minute renders as less-than-one`() {
        assertEquals("<1m", 30.seconds.formatHuman())
    }

    @Test
    fun `between one and sixty minutes renders as Xm`() {
        assertEquals("42m", 42.minutes.formatHuman())
    }

    @Test
    fun `whole hours render without minutes`() {
        assertEquals("3h", 3.hours.formatHuman())
    }

    @Test
    fun `hours and minutes render with both components`() {
        assertEquals("3h 12m", (3.hours + 12.minutes).formatHuman())
    }
}
