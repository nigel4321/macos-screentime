package com.nigel4321.screentime.feature.dashboard.today.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.feature.dashboard.today.formatHuman
import kotlin.time.Duration

private const val MAX_ROWS = 5

@Composable
internal fun TopAppsTile(
    rows: List<UsageRow>,
    modifier: Modifier = Modifier,
) {
    BentoTile(modifier = modifier) {
        Column(modifier = Modifier.fillMaxWidth()) {
            Text(
                text = "Top apps",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(12.dp))

            val visible = rows.take(MAX_ROWS)
            val maxDuration = visible.firstOrNull()?.duration ?: Duration.ZERO
            visible.forEachIndexed { index, row ->
                if (index > 0) Spacer(Modifier.height(10.dp))
                AppBarRow(row = row, maxDuration = maxDuration)
            }
        }
    }
}

/**
 * One horizontal bar per app: label on the left, growing bar in the
 * middle, formatted duration on the right. Hand-rolled for now — fine
 * for 5 rows and avoids the Vico learning curve. Can graduate to a
 * proper `CartesianChartHost` in §2.18 polish.
 */
@Composable
private fun AppBarRow(
    row: UsageRow,
    maxDuration: Duration,
) {
    val ratio =
        if (maxDuration > Duration.ZERO) {
            (row.duration.inWholeSeconds.toFloat() / maxDuration.inWholeSeconds.toFloat())
                .coerceIn(MIN_BAR_FRACTION, 1f)
        } else {
            MIN_BAR_FRACTION
        }

    Column(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = row.bundleId?.value ?: "Unknown",
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.weight(1f),
                maxLines = 1,
            )
            Spacer(Modifier.width(8.dp))
            Text(
                text = row.duration.formatHuman(),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Spacer(Modifier.height(4.dp))
        Box(
            modifier =
                Modifier
                    .fillMaxWidth()
                    .height(6.dp)
                    .background(
                        color = MaterialTheme.colorScheme.surfaceContainerHigh,
                        shape = RoundedCornerShape(3.dp),
                    ),
        ) {
            Box(
                modifier =
                    Modifier
                        .fillMaxWidth(ratio)
                        .height(6.dp)
                        .padding(end = 0.dp)
                        .background(
                            color = MaterialTheme.colorScheme.primary,
                            shape = RoundedCornerShape(3.dp),
                        ),
            )
        }
    }
}

// 4% min so a tiny duration is still visually present, not invisible.
private const val MIN_BAR_FRACTION = 0.04f
