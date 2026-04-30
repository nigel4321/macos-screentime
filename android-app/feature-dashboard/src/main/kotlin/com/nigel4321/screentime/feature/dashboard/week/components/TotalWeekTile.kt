package com.nigel4321.screentime.feature.dashboard.week.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.nigel4321.screentime.feature.dashboard.today.components.BentoTile
import com.nigel4321.screentime.feature.dashboard.today.formatHuman
import kotlin.time.Duration

@Composable
internal fun TotalWeekTile(
    total: Duration,
    modifier: Modifier = Modifier,
) {
    BentoTile(modifier = modifier) {
        Column {
            Text(
                text = "Total this week",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Spacer(Modifier.height(8.dp))
            Text(
                text = total.formatHuman(),
                style = MaterialTheme.typography.displaySmall,
                modifier =
                    Modifier.semantics {
                        contentDescription = "Total usage this week: ${total.formatHuman()}"
                    },
            )
        }
    }
}
