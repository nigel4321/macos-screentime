package com.nigel4321.screentime.feature.dashboard.week.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.nigel4321.screentime.feature.dashboard.today.formatHuman
import com.nigel4321.screentime.feature.dashboard.week.DayBucket
import java.time.format.DateTimeFormatter
import kotlin.time.Duration

/**
 * Hand-rolled bar chart for the 7 daily totals. We could swap this out
 * for `vico-compose-m3`'s `CartesianChartHost` once we want stacked
 * bars (top apps per day) — for the single-series MVP a few `Box`es
 * handle it without the library's learning curve.
 */
@Composable
internal fun WeekBarChart(
    days: List<DayBucket>,
    modifier: Modifier = Modifier,
) {
    val maxDuration = days.maxOfOrNull { it.duration } ?: Duration.ZERO

    // The enclosing [WeekChartTile] supplies a merged semantics
    // description that already includes every day + duration, so the
    // chart itself stays unannounced — TalkBack reads the tile as one
    // unit instead of an axis-by-axis traversal.
    Column(
        modifier =
            modifier
                .fillMaxWidth()
                .heightIn(min = MIN_CHART_HEIGHT),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth().weight(1f),
            horizontalArrangement = Arrangement.spacedBy(BAR_GAP, Alignment.CenterHorizontally),
            verticalAlignment = Alignment.Bottom,
        ) {
            days.forEach { day ->
                Bar(
                    fraction = day.duration.fractionOf(maxDuration),
                    modifier = Modifier.weight(1f).fillMaxHeight(),
                )
            }
        }
        Spacer(Modifier.height(6.dp))
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(BAR_GAP, Alignment.CenterHorizontally),
        ) {
            days.forEach { day ->
                Text(
                    text = day.day.format(DAY_INITIAL),
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun Bar(
    fraction: Float,
    modifier: Modifier = Modifier,
) {
    Box(
        modifier = modifier.padding(horizontal = 2.dp),
        contentAlignment = Alignment.BottomCenter,
    ) {
        // Grey track behind so empty days still show a visible column.
        Box(
            modifier =
                Modifier
                    .fillMaxSize()
                    .background(
                        color = MaterialTheme.colorScheme.surfaceContainerHigh,
                        shape = RoundedCornerShape(BAR_RADIUS),
                    ),
        )
        // Foreground bar only when there's actual usage. A no-data day
        // shows the grey track on its own, which is the desired
        // "nothing recorded" affordance.
        if (fraction > 0f) {
            Box(
                modifier =
                    Modifier
                        .fillMaxWidth()
                        .fillMaxHeight(fraction.coerceAtLeast(MIN_BAR_FRACTION))
                        .background(
                            color = MaterialTheme.colorScheme.primary,
                            shape = RoundedCornerShape(BAR_RADIUS),
                        ),
            )
        }
    }
}

private fun Duration.fractionOf(max: Duration): Float =
    if (max > Duration.ZERO) (inWholeSeconds.toFloat() / max.inWholeSeconds.toFloat()).coerceIn(0f, 1f) else 0f

/**
 * Spoken summary of the seven daily totals in the chart, used by
 * [WeekChartTile] for the tile-level TalkBack label. Visible to that
 * sibling Composable only.
 */
internal fun chartDescription(days: List<DayBucket>): String =
    days.joinToString(separator = "; ") { "${it.day.format(DAY_FULL)}: ${it.duration.formatHuman()}" }

// 4% min so a sub-minute day still shows something rather than vanishing
// into the track. 0% on a no-data day would look identical to the grey
// track, which is the desired "nothing recorded" affordance.
private const val MIN_BAR_FRACTION = 0.04f
private val BAR_GAP = 8.dp
private val BAR_RADIUS = 4.dp
private val MIN_CHART_HEIGHT = 140.dp
private val DAY_INITIAL = DateTimeFormatter.ofPattern("EEE")
private val DAY_FULL = DateTimeFormatter.ofPattern("EEEE d MMM")
