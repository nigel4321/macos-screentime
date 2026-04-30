package com.nigel4321.screentime.feature.dashboard.week.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.nigel4321.screentime.feature.dashboard.today.components.BentoTile
import com.nigel4321.screentime.feature.dashboard.week.DayBucket

@Composable
internal fun WeekChartTile(
    days: List<DayBucket>,
    modifier: Modifier = Modifier,
) {
    BentoTile(
        modifier =
            modifier.semantics(mergeDescendants = true) {
                contentDescription = "Daily totals: ${chartDescription(days)}"
            },
    ) {
        Column(modifier = Modifier.fillMaxWidth()) {
            Text(
                text = "Daily totals",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(12.dp))
            WeekBarChart(days = days)
        }
    }
}
