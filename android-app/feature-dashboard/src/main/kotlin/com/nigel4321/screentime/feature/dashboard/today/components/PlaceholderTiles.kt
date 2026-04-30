package com.nigel4321.screentime.feature.dashboard.today.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/**
 * Two intentionally-stubbed tiles holding their bento-grid slots so the
 * layout shape stays correct even without the supporting data.
 *
 * - [CategoriesTile]: the backend doesn't aggregate by category yet
 *   (roadmap §4.1). Placeholder copy makes the gap visible.
 * - [DowntimeStatusTile]: policy persistence lands in §3.7. Until then
 *   we render the "no active downtime" steady state.
 */
@Composable
internal fun CategoriesTile(modifier: Modifier = Modifier) {
    BentoTile(modifier = modifier) {
        Column {
            Text(
                text = "Categories",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = "Coming with category aggregation in §4.1",
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}

@Composable
internal fun DowntimeStatusTile(modifier: Modifier = Modifier) {
    BentoTile(modifier = modifier) {
        Column {
            Text(
                text = "Downtime",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = "No active downtime",
                style = MaterialTheme.typography.bodyMedium,
            )
        }
    }
}
