package com.nigel4321.screentime.feature.dashboard.today.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.nigel4321.screentime.core.domain.model.UsageRow
import com.nigel4321.screentime.feature.dashboard.today.formatHuman
import kotlin.time.Duration

/**
 * One bento cell per app — half-width on the 2-col grid. Used for the
 * top-N apps row(s) under "Total today".
 *
 * Layout:
 *   - rank  • bundle id (or display name) on top, ellipsised
 *   - duration in the upper-bigger style
 *   - thin progress bar at the bottom showing this app's share of the
 *     leader's duration, so the eye can compare across tiles
 *
 * `maxDuration` is the duration of the leader (rank 1). Pass the same
 * value to every tile in a row so the bars are comparable.
 */
@Composable
internal fun TopAppTile(
    row: UsageRow,
    maxDuration: Duration,
    rank: Int,
    modifier: Modifier = Modifier,
) {
    // Prefer the server-supplied display name (e.g. "Google Chrome");
    // fall back to the bundle id ("com.google.Chrome") when metadata
    // hasn't reached the backend yet, and to "Unknown" if even that's
    // missing.
    val name =
        row.displayName?.takeIf { it.isNotBlank() }
            ?: row.bundleId?.value
            ?: "Unknown"
    val ratio =
        if (maxDuration > Duration.ZERO) {
            (row.duration.inWholeSeconds.toFloat() / maxDuration.inWholeSeconds.toFloat())
                .coerceIn(MIN_BAR_FRACTION, 1f)
        } else {
            MIN_BAR_FRACTION
        }

    BentoTile(
        modifier = modifier.semantics { contentDescription = "Rank $rank: $name, ${row.duration.formatHuman()}" },
    ) {
        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.SpaceBetween,
        ) {
            TileHeader(rank = rank, name = name)
            Spacer(Modifier.height(8.dp))
            Text(
                text = row.duration.formatHuman(),
                style = MaterialTheme.typography.headlineSmall,
            )
            Spacer(Modifier.height(8.dp))
            ProgressBar(ratio = ratio)
        }
    }
}

@Composable
private fun TileHeader(
    rank: Int,
    name: String,
) {
    Column {
        Text(
            text = "#$rank",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Spacer(Modifier.height(2.dp))
        Text(
            text = name,
            style = MaterialTheme.typography.titleSmall,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis,
        )
    }
}

@Composable
private fun ProgressBar(ratio: Float) {
    Box(
        modifier =
            Modifier
                .fillMaxWidth()
                .height(BAR_HEIGHT)
                .background(
                    color = MaterialTheme.colorScheme.surfaceContainerHigh,
                    shape = RoundedCornerShape(BAR_RADIUS),
                ),
    ) {
        Box(
            modifier =
                Modifier
                    .fillMaxWidth(ratio)
                    .height(BAR_HEIGHT)
                    .background(
                        color = MaterialTheme.colorScheme.primary,
                        shape = RoundedCornerShape(BAR_RADIUS),
                    ),
        )
    }
}

// 4% min so a tiny duration is still visually present, not invisible.
private const val MIN_BAR_FRACTION = 0.04f
private val BAR_HEIGHT = 4.dp
private val BAR_RADIUS = 2.dp
