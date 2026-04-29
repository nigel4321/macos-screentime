package com.nigel4321.screentime.core.data.api

import com.nigel4321.screentime.core.data.api.dto.DeviceListResponse
import com.nigel4321.screentime.core.data.api.dto.GoogleAuthRequest
import com.nigel4321.screentime.core.data.api.dto.PairCompleteRequest
import com.nigel4321.screentime.core.data.api.dto.PolicyResponse
import com.nigel4321.screentime.core.data.api.dto.SummaryResponse
import com.nigel4321.screentime.core.data.api.dto.TokenResponse
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Query

interface ScreentimeApi {
    @POST("v1/auth/google")
    suspend fun authGoogle(
        @Body request: GoogleAuthRequest,
    ): TokenResponse

    @POST("v1/account:pair-complete")
    suspend fun pairComplete(
        @Body request: PairCompleteRequest,
    ): TokenResponse

    @GET("v1/usage:summary")
    suspend fun usageSummary(
        @Query("from") from: String,
        @Query("to") to: String,
        @Query("groupBy") groupBy: String? = null,
    ): SummaryResponse

    @GET("v1/policy/current")
    suspend fun currentPolicy(): PolicyResponse

    @GET("v1/devices")
    suspend fun listDevices(): DeviceListResponse
}
