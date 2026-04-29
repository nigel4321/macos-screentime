package com.nigel4321.screentime.feature.onboarding.auth

import android.app.Activity
import android.content.Context
import androidx.credentials.CredentialManager
import androidx.credentials.CustomCredential
import androidx.credentials.GetCredentialRequest
import androidx.credentials.exceptions.GetCredentialException
import com.google.android.libraries.identity.googleid.GetGoogleIdOption
import com.google.android.libraries.identity.googleid.GoogleIdTokenCredential
import com.nigel4321.screentime.feature.onboarding.BuildConfig
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class CredentialManagerGoogleSignInClient
    @Inject
    constructor(
        @ApplicationContext context: Context,
    ) : GoogleSignInClient {
        private val credentialManager = CredentialManager.create(context)

        override suspend fun requestSignIn(activity: Activity): Result<String> {
            val webClientId = BuildConfig.GOOGLE_WEB_CLIENT_ID
            if (webClientId.isBlank()) {
                return Result.failure(MissingGoogleWebClientIdException())
            }

            // filterByAuthorizedAccounts=false on first sign-in so the user
            // sees Google's account picker; the second pass typically uses
            // true but we leave the simpler one-shot flow for now.
            val googleIdOption =
                GetGoogleIdOption.Builder()
                    .setFilterByAuthorizedAccounts(false)
                    .setServerClientId(webClientId)
                    .setAutoSelectEnabled(false)
                    .build()
            val request =
                GetCredentialRequest.Builder()
                    .addCredentialOption(googleIdOption)
                    .build()

            return runCatching {
                val response = credentialManager.getCredential(activity, request)
                val credential = response.credential
                require(
                    credential is CustomCredential &&
                        credential.type == GoogleIdTokenCredential.TYPE_GOOGLE_ID_TOKEN_CREDENTIAL,
                ) {
                    "Unexpected credential type: ${credential::class.java.name}"
                }
                GoogleIdTokenCredential.createFrom(credential.data).idToken
            }.recoverCatching { error ->
                // Surface Credential Manager errors with their cause so the
                // ViewModel can render a useful message.
                if (error is GetCredentialException) {
                    throw IllegalStateException(error.localizedMessage ?: error.type, error)
                }
                throw error
            }
        }
    }
