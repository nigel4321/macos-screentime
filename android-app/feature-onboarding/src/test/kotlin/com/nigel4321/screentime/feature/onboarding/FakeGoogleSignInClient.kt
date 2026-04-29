package com.nigel4321.screentime.feature.onboarding

import android.app.Activity
import com.nigel4321.screentime.feature.onboarding.auth.GoogleSignInClient
import kotlinx.coroutines.CompletableDeferred

internal class FakeGoogleSignInClient : GoogleSignInClient {
    private var nextResult: Result<String>? = null
    val started = CompletableDeferred<Unit>()

    fun returns(result: Result<String>) {
        nextResult = result
    }

    override suspend fun requestSignIn(activity: Activity): Result<String> {
        started.complete(Unit)
        return checkNotNull(nextResult) { "FakeGoogleSignInClient.returns() not called" }
    }
}
