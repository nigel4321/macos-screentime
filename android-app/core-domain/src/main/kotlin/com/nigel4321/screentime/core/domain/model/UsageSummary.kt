package com.nigel4321.screentime.core.domain.model

import java.time.LocalDate
import kotlin.time.Duration

data class UsageSummary(
    val rows: List<UsageRow>,
)

data class UsageRow(
    val bundleId: BundleId?,
    val day: LocalDate?,
    val duration: Duration,
    /**
     * Server-supplied human display name (`com.google.Chrome` →
     * "Google Chrome"). Source of truth is the Mac agent's
     * `AppMetadata` resolver (§1.12), uploaded via the §2.22 backend
     * `app_metadata` catalog. Null when no metadata row exists for the
     * `(account, bundle_id)` pair — UI falls back to [bundleId].
     */
    val displayName: String? = null,
)
