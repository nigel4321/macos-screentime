package com.nigel4321.screentime.feature.onboarding.auth

import android.app.Activity

/**
 * Wraps the Credential Manager call-site so [OnboardingViewModel] is
 * testable without an Activity. The real implementation lives in
 * [CredentialManagerGoogleSignInClient] and is bound by Hilt for
 * production builds.
 */
interface GoogleSignInClient {
    suspend fun requestSignIn(activity: Activity): Result<String>
}

/** Thrown when [BuildConfig.GOOGLE_WEB_CLIENT_ID] is empty at runtime. */
class MissingGoogleWebClientIdException :
    IllegalStateException(
        "GOOGLE_WEB_CLIENT_ID is not set. Provide it via the " +
            "SCREENTIME_GOOGLE_WEB_CLIENT_ID env var or " +
            "screentime.googleWebClientId Gradle property and rebuild.",
    )
