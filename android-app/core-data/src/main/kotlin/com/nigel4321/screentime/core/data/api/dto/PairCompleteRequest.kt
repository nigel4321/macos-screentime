package com.nigel4321.screentime.core.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class PairCompleteRequest(
    @SerialName("code") val code: String,
)
