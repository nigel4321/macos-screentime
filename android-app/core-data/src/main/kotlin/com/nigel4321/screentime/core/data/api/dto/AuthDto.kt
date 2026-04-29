package com.nigel4321.screentime.core.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class GoogleAuthRequest(
    @SerialName("id_token") val idToken: String,
)

@Serializable
data class TokenResponse(
    @SerialName("token") val token: String,
    @SerialName("expires_at") val expiresAt: String,
)
